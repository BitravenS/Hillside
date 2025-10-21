package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"hillside/internal/models"
	"hillside/internal/utils"
)

type HistoryManager struct {
	// write queue and worker control
	writeQ chan messageWriteRequest
	wg     sync.WaitGroup
	stopCh chan struct{}

	// in-memory caches (last chain index per room)
	lastIndexMu sync.RWMutex
	lastIndex   map[string]uint64

	writeBatchSize int           // how many envelopes to write in a single transaction
	writeFlushFreq time.Duration // max wait before flushing batch
}

type messageWriteRequest struct {
	storedMsg models.StoredMessage
	ctx       context.Context
	result    chan error
}

type CatchUpMessages struct {
	ReturnedMessages []models.StoredMessage
	SenderID         string
}

// NewHistoryManager returns a ready-to-start history manager.
// store: your SQLite or PG store implementing Store.
// writeQSize: buffered channel size for incoming writes.
func NewHistoryManager(writeQSize int) *HistoryManager {
	h := &HistoryManager{
		writeQ:         make(chan messageWriteRequest, writeQSize),
		stopCh:         make(chan struct{}),
		lastIndex:      make(map[string]uint64),
		writeBatchSize: 50,
		writeFlushFreq: 200 * time.Millisecond,
	}
	return h
}

// Start launches the background writer. Call Stop() to cleanly shut down.
func (h *HistoryManager) Start(store *Store) {
	h.wg.Add(1)
	go h.writeWorker(store)
}

// Stop stops worker and waits for it to finish. It blocks until writer drained.
func (h *HistoryManager) Stop() {
	close(h.stopCh)
	h.wg.Wait()
}

func (h *HistoryManager) EnqueueEnvelope(
	ctx context.Context,
	signature, payload []byte,
	timestamp int64,
	msgType models.MessageType,
	chainIndex *uint64,
	sender_id, roomID, serverID string,
) error {

	req := messageWriteRequest{
		storedMsg: models.StoredMessage{
			RoomID:     roomID,
			ServerID:   serverID,
			ChainIndex: chainIndex,
			MsgType:    msgType,
			SenderID:   sender_id,
			Timestamp:  timestamp,
			Signature:  signature,
			Payload:    payload,
		},
		ctx:    ctx,
		result: make(chan error, 1),
	}

	select {
	case h.writeQ <- req:
		// enqueued
		return nil
	default:
		// queue full: fail fast to caller
		return errors.New("history write queue full")
	}
}

// writeWorker batches writes into the DB to limit transactions and contention.
func (h *HistoryManager) writeWorker(store *Store) {
	defer h.wg.Done()
	batch := make([]messageWriteRequest, 0, h.writeBatchSize)
	flushTimer := time.NewTimer(h.writeFlushFreq)
	defer flushTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		for _, r := range batch {
			_ = r.ctx // currently unused, but could use store.WithContext
			if err := store.SaveEnvelope(context.Background(), r.storedMsg.Signature, r.storedMsg.Payload, r.storedMsg.Timestamp, r.storedMsg.MsgType, r.storedMsg.ChainIndex, r.storedMsg.SenderID, r.storedMsg.RoomID, r.storedMsg.ServerID); err != nil {
				log.Printf("history: save envelope error: %v", err)
				r.result <- err
			} else {
				// update lastIndex cache if chain present
				if r.storedMsg.ChainIndex != nil && r.storedMsg.RoomID != "" {
					h.setLastIndex(r.storedMsg.RoomID, *r.storedMsg.ChainIndex)
				}
				r.result <- nil
			}
			close(r.result)
		}
		// reset batch
		batch = batch[:0]
	}

	for {
		select {
		case <-h.stopCh:
			// drain queue before exiting
			for {
				select {
				case req := <-h.writeQ:
					batch = append(batch, req)
					if len(batch) >= h.writeBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case req := <-h.writeQ:
			batch = append(batch, req)
			if len(batch) >= h.writeBatchSize {
				flush()
				if !flushTimer.Stop() {
					<-flushTimer.C
				}
				flushTimer.Reset(h.writeFlushFreq)
			}
		case <-flushTimer.C:
			flush()
			flushTimer.Reset(h.writeFlushFreq)
		}
	}
}

// setLastIndex updates in-memory lastIndex
func (h *HistoryManager) setLastIndex(room string, idx uint64) {
	h.lastIndexMu.Lock()
	defer h.lastIndexMu.Unlock()
	if existing, ok := h.lastIndex[room]; !ok || idx > existing {
		h.lastIndex[room] = idx
	}
}

// GetLastIndex returns cached last chain index if available, else ErrNoRows.
func (h *HistoryManager) GetLastIndex(room string, store *Store) (uint64, error) {
	h.lastIndexMu.RLock()
	defer h.lastIndexMu.RUnlock()
	if v, ok := h.lastIndex[room]; ok {
		return v, nil
	}
	// fallback to DB
	idx, err := store.GetLatestChainIndex(context.Background(), room)
	if err != nil {
		return 0, err
	}
	// cache it
	h.setLastIndex(room, idx)
	return idx, nil
}

// BuildCatchUpPayload will fetch messages since `sinceIndex`, compress them to a gzipped
func (h *HistoryManager) BuildCatchUpPayload(ctx context.Context, roomID string, sinceIndex uint64, limit int, store *Store) (payload []byte, lastIndex uint64, err error, msgs []models.StoredMessage) {
	msgs, err = store.GetMessagesSinceChainIndex(ctx, roomID, sinceIndex, limit)
	if err != nil {
		return nil, 0, err, nil
	}
	if len(msgs) == 0 {
		latest, _ := h.GetLastIndex(roomID, store)
		return nil, latest, nil, nil
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	for _, m := range msgs {
		entry, err := json.Marshal(m)
		if err != nil {
			return nil, 0, err, nil
		}
		if err := writeFrame(gw, entry); err != nil {
			_ = gw.Close()
			return nil, 0, err, nil
		}
		lastIndex = 0
		if m.ChainIndex != nil {
			lastIndex = *m.ChainIndex
		}
	}
	if err := gw.Close(); err != nil {
		return nil, 0, err, nil
	}
	return buf.Bytes(), lastIndex, nil, msgs
}

var RL, _ = utils.NewRemoteLogger(7000)

// DecompressCatchUpPayload decompresses the payload and writes the entries to the db.
func (h *HistoryManager) DecompressCatchUpPayload(ctx context.Context, payload []byte, roomID string, store *Store) (*CatchUpMessages, error) {
	if len(payload) == 0 {
		return &CatchUpMessages{ReturnedMessages: make([]models.StoredMessage, 0)}, nil
	}
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	RL.Logf("Created gzip object")
	if err != nil {
		RL.Logf("Got an error at 1: %+v", err)
		return nil, err
	}
	catchUpMsgs := &CatchUpMessages{
		ReturnedMessages: make([]models.StoredMessage, 0),
	}
	defer gr.Close()

	for {
		entry, err := readFrame(gr)
		if err != nil {

			RL.Logf("Got an error at 2: %+v", err)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {

				RL.Logf("ERROR IS EOF: %+v", err)
				break
			}

			RL.Logf("Returning with unhandled error")
			return nil, err
		}
		var dec *models.StoredMessage
		err = json.Unmarshal(entry, &dec)
		if err != nil {

			RL.Logf("Got an error at 3: %+v", err)
			return nil, err
		}
		catchUpMsg := models.StoredMessage{
			RoomID:     dec.RoomID,
			ServerID:   dec.ServerID,
			Payload:    dec.Payload,
			Signature:  dec.Signature,
			ChainIndex: dec.ChainIndex,
			SenderID:   dec.SenderID,
			Timestamp:  dec.Timestamp,
			MsgType:    dec.MsgType,
		}
		catchUpMsgs.ReturnedMessages = append(catchUpMsgs.ReturnedMessages, catchUpMsg)
	}

	RL.Logf("Returning the correct way with")
	return catchUpMsgs, nil
}

/*
func (h *HistoryManager) BuildEncryptedCatchUpResponse(ctx context.Context, roomID string, sinceIndex uint64, limit int, recipientPubKey []byte) (encPayload []byte, lastIndex uint64, err error) {
	payload, lastIndex, err := h.BuildCatchUpPayload(ctx, roomID, sinceIndex, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(payload) == 0 {
		return nil, lastIndex, nil
	}

	// TODO: replace with your project's encryption API. Example:
	// enc, err := crypto.SealWithPublicKey(recipientPubKey, payload)
	// return enc, lastIndex, err
	enc, err := crypto.SealFor(recipientPubKey, payload) // implement SealFor or equivalent
	if err != nil {
		return nil, 0, err
	}
	return enc, lastIndex, nil
}
*/

// SaveEnvelope stores an envelope. Use chainIndex != nil for chat messages.
// Behavior: insert is idempotent (duplicate chain_index for same room ignored).
func (s *Store) SaveEnvelope(ctx context.Context, signature, payload []byte, timestamp int64, msgType models.MessageType, chainIndex *uint64, sender_id, roomID, serverID string) error {
	var ci any
	if chainIndex != nil {
		ci = int64(*chainIndex)
	} else {
		ci = nil
	}

	const q = `
INSERT OR IGNORE INTO messages
(room_id, server_id, chain_index, msg_type, sender_id, timestamp, signature, payload)
VALUES (?, ?, ?, ?, ?, ?, ?,?);
`
	_, err := s.db.ExecContext(ctx, q,
		roomID,
		serverID,
		ci,
		string(msgType),
		sender_id,
		timestamp,
		signature,
		payload,
	)
	if err != nil {
		return fmt.Errorf("insert envelope: %w", err)
	}
	return nil
}

// GetMessagesSinceChainIndex returns chat messages with chain_index > sinceIndex ordered ASC.
func (s *Store) GetMessagesSinceChainIndex(ctx context.Context, roomID string, sinceIndex uint64, limit int) ([]models.StoredMessage, error) {
	var q = `
SELECT id, room_id, server_id, chain_index, msg_type, sender_id, timestamp, signature, payload
FROM messages
WHERE room_id = ? AND chain_index IS NOT NULL AND chain_index >= ?
ORDER BY chain_index ASC
`
	var rows *sql.Rows
	var err error
	if limit <= 0 {
		q += ";"
		rows, err = s.db.QueryContext(ctx, q, roomID, int64(sinceIndex))
	} else {
		q += " LIMIT ?;"
		rows, err = s.db.QueryContext(ctx, q, roomID, int64(sinceIndex), limit)
	}
	if err != nil {
		return nil, fmt.Errorf("select messages since: %w", err)
	}
	defer rows.Close()

	var out []models.StoredMessage
	for rows.Next() {
		var (
			id        int64
			room      string
			server    sql.NullString
			chainN    sql.NullInt64
			msgType   string
			senderID  sql.NullString
			timestamp int64
			signature []byte
			payload   []byte
		)
		if err := rows.Scan(&id, &room, &server, &chainN, &msgType, &senderID, &timestamp, &signature, &payload); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		var ci *uint64
		if chainN.Valid {
			v := uint64(chainN.Int64)
			ci = &v
		}
		sm := models.StoredMessage{
			ID:         id,
			RoomID:     room,
			ServerID:   server.String,
			ChainIndex: ci,
			MsgType:    models.MessageType(msgType),
			SenderID:   senderID.String,
			Timestamp:  timestamp,
			Signature:  signature,
			Payload:    payload,
		}
		out = append(out, sm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetLatestMessages returns latest messages ordered like Postgres implementation.
func (s *Store) GetLatestMessages(ctx context.Context, roomID string, limit int) ([]models.StoredMessage, error) {
	const q = `
SELECT id, room_id, server_id, chain_index, msg_type, sender_id, timestamp, signature, payload
FROM messages
WHERE room_id = ?
ORDER BY
  CASE WHEN chain_index IS NOT NULL THEN 0 ELSE 1 END,
  chain_index ASC,
  timestamp ASC
LIMIT ?;
`
	rows, err := s.db.QueryContext(ctx, q, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("select latest messages: %w", err)
	}
	defer rows.Close()

	var out []models.StoredMessage
	for rows.Next() {
		var (
			id        int64
			room      string
			server    sql.NullString
			chainN    sql.NullInt64
			msgType   string
			senderID  sql.NullString
			timestamp int64
			signature []byte
			payload   []byte
		)
		if err := rows.Scan(&id, &room, &server, &chainN, &msgType, &senderID, &timestamp, &signature, &payload); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		var ci *uint64
		if chainN.Valid {
			v := uint64(chainN.Int64)
			ci = &v
		}
		sm := models.StoredMessage{
			ID:         id,
			RoomID:     room,
			ServerID:   server.String,
			ChainIndex: ci,
			MsgType:    models.MessageType(msgType),
			SenderID:   senderID.String,
			Timestamp:  timestamp,
			Signature:  signature,
			Payload:    payload,
		}
		out = append(out, sm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetLatestChainIndex returns highest chain_index for room or ErrNoRows.
func (s *Store) GetLatestChainIndex(ctx context.Context, roomID string) (uint64, error) {
	const q = `
SELECT chain_index FROM messages
WHERE room_id = ? AND chain_index IS NOT NULL
ORDER BY chain_index DESC
LIMIT 1;
`
	var chain sql.NullInt64
	if err := s.db.QueryRowContext(ctx, q, roomID).Scan(&chain); err != nil {
		if errors.Is(err, sql.ErrNoRows) || !chain.Valid {
			return 0, ErrNoRows
		}
		return 0, fmt.Errorf("select latest chain index: %w", err)
	}
	if !chain.Valid {
		return 0, ErrNoRows
	}
	return uint64(chain.Int64), nil
}

// DeleteOlderThan deletes messages older than `before` and returns rows deleted.
func (s *Store) DeleteOlderThan(ctx context.Context, roomID string, before time.Time) (int64, error) {
	const q = `
DELETE FROM messages WHERE room_id = ? AND timestamp < ?;
`
	res, err := s.db.ExecContext(ctx, q, roomID, before.UnixMicro())
	if err != nil {
		return 0, fmt.Errorf("delete older than: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
