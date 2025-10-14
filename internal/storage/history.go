package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"hillside/internal/models"
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
	raw      []byte
	env      *models.Envelope
	chain    *uint64
	roomID   string
	serverID string
	ctx      context.Context
	result   chan error
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

func (h *HistoryManager) EnqueueEnvelope(ctx context.Context, raw []byte, env *models.Envelope, msg models.Message, roomID string, serverID string) error {
	var chainIdx uint64
	switch env.Type {
	case models.MsgTypeChat:
		m := msg.(*models.ChatMessage)
		chainIdx = m.ChainIndex
	default:
		// control messages may carry room info in payload; leave roomID empty here
	}

	req := messageWriteRequest{
		raw:      raw,
		env:      env,
		chain:    &chainIdx,
		roomID:   roomID,
		serverID: serverID,
		ctx:      ctx,
		result:   make(chan error, 1),
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
			if err := store.SaveEnvelope(context.Background(), r.raw, r.env, r.chain, r.roomID, r.serverID); err != nil {
				log.Printf("history: save envelope error: %v", err)
				r.result <- err
			} else {
				// update lastIndex cache if chain present
				if r.chain != nil && r.roomID != "" {
					h.setLastIndex(r.roomID, *r.chain)
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
func (h *HistoryManager) BuildCatchUpPayload(ctx context.Context, roomID string, sinceIndex uint64, limit int, store *Store) (payload []byte, lastIndex uint64, err error) {
	msgs, err := store.GetMessagesSinceChainIndex(ctx, roomID, sinceIndex, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(msgs) == 0 {
		latest, _ := h.GetLastIndex(roomID, store)
		return nil, latest, nil
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	for _, m := range msgs {
		if err := writeFrame(gw, m.Signature); err != nil {
			_ = gw.Close()
			return nil, 0, err
		}
		lastIndex = 0
		if m.ChainIndex != nil {
			lastIndex = *m.ChainIndex
		}
	}
	if err := gw.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), lastIndex, nil
}

// Helper: write framed signature into gzip writer (8-byte length + data)

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
