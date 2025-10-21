package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"hillside/internal/models"
	// "hillside/internal/utils"
)

// var RL, _ = utils.NewRemoteLogger(5000)

type PeerManager struct {
	// write queue and worker control
	writeQ chan userWriteRequest
	wg     sync.WaitGroup
	stopCh chan struct{}

	writeBatchSize int           // how many envelopes to write in a single transaction
	writeFlushFreq time.Duration // max wait before flushing batch
}

type userWriteRequest struct {
	user   *models.User
	ctx    context.Context
	result chan error
}

func NewPeerManager(writeQSize int) *PeerManager {
	p := &PeerManager{
		writeQ:         make(chan userWriteRequest, writeQSize),
		stopCh:         make(chan struct{}),
		writeBatchSize: 1,
		writeFlushFreq: 200 * time.Millisecond,
	}
	// RL.Logf("PeerManager initialized with writeQSize=%d", writeQSize)
	return p
}

func (p *PeerManager) Start(store *Store) {
	p.wg.Add(1)
	go p.peerWriteWorker(store)
}

// Stop stops worker and waits for it to finisp. It blocks until writer drained.
func (p *PeerManager) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *PeerManager) EnqueueUserEntry(ctx context.Context, user *models.User) error {

	req := userWriteRequest{
		user:   user,
		ctx:    ctx,
		result: make(chan error, 1),
	}
	// RL.Logf("Enqueueing user: %v", user.PeerID)

	select {
	case p.writeQ <- req:
		return nil
	default:
		return errors.New("peer manager write queue full")
	}
}

func (s *Store) SaveUser(ctx context.Context, user *models.User) error {
	pid := user.PeerID
	dilithiumPub := user.DilithiumPub
	kyberPub := user.KyberPub
	libpub := user.Libp2pPub
	name := user.Username
	color := user.PreferredColor
	lastSeen := time.Now().UnixMicro()

	const q = `
	INSERT OR REPLACE INTO peers
	(peer_id, dilithium_pub, kyber_pub, libp2p_pub, username, color, last_seen, synced)
	VALUES (?, ?, ?, ?, ?, ?,?,?);
`
	_, err := s.db.ExecContext(ctx, q,
		pid,
		dilithiumPub,
		kyberPub,
		libpub,
		name,
		color,
		lastSeen,
		1,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *Store) GetUserByID(ctx context.Context, peerID string) (*models.User, error) {
	const q = `
SELECT peer_id, dilithium_pub, kyber_pub, libp2p_pub, username, color, last_seen, synced
	FROM peers
	WHERE peer_id = ?
	LIMIT 1;
`
	rows, err := s.db.QueryContext(ctx, q, peerID)
	if err != nil {
		return nil, fmt.Errorf("select user by id: %w", err)
	}

	defer rows.Close()
	var out *models.User
	for rows.Next() {
		var (
			peerID       string
			dilithiumPub []byte
			kyberPub     []byte
			libp2pPub    []byte
			username     sql.NullString
			color        sql.NullString
			lastSeen     sql.NullInt64
			synced       sql.NullInt64
		)
		if err := rows.Scan(&peerID, &dilithiumPub, &kyberPub, &libp2pPub, &username, &color, &lastSeen, &synced); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = &models.User{
			PeerID:         peerID,
			DilithiumPub:   dilithiumPub,
			KyberPub:       kyberPub,
			Libp2pPub:      libp2pPub,
			Username:       username.String,
			PreferredColor: color.String,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	const q = `
	SELECT peer_id, dilithium_pub, kyber_pub, libp2p_pub, username, color, last_seen, synced
	FROM peers;
`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("select all users: %w", err)
	}

	defer rows.Close()
	var out []*models.User
	for rows.Next() {
		var (
			id           int64
			peerID       string
			dilithiumPub []byte
			kyberPub     []byte
			libp2pPub    []byte
			username     sql.NullString
			color        sql.NullString
			lastSeen     sql.NullInt64
			synced       sql.NullInt64
		)
		if err := rows.Scan(&id, &peerID, &dilithiumPub, &kyberPub, &libp2pPub, &username, &color, &lastSeen, &synced); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		m := &models.User{
			PeerID:         peerID,
			DilithiumPub:   dilithiumPub,
			KyberPub:       kyberPub,
			Libp2pPub:      libp2pPub,
			Username:       username.String,
			PreferredColor: color.String,
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetLastSeenUsers(ctx context.Context, since time.Time) ([]*models.User, error) {
	sinceUnix := since.UnixMicro()
	const q = `
SELECT peer_id, dilithium_pub, kyber_pub, libp2p_pub, username, color, last_seen, synced
	FROM peers
	WHERE last_seen >= ?;
`
	rows, err := s.db.QueryContext(ctx, q, sinceUnix)
	if err != nil {
		return nil, fmt.Errorf("select users by last seen: %w", err)
	}
	defer rows.Close()
	var out []*models.User
	for rows.Next() {
		var (
			id           int64
			peerID       string
			dilithiumPub []byte
			kyberPub     []byte
			libp2pPub    []byte
			username     sql.NullString
			color        sql.NullString
			lastSeen     sql.NullInt64
			synced       sql.NullInt64
		)
		if err := rows.Scan(&id, &peerID, &dilithiumPub, &kyberPub, &libp2pPub, &username, &color, &lastSeen, &synced); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		m := &models.User{
			PeerID:         peerID,
			DilithiumPub:   dilithiumPub,
			KyberPub:       kyberPub,
			Libp2pPub:      libp2pPub,
			Username:       username.String,
			PreferredColor: color.String,
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *PeerManager) peerWriteWorker(store *Store) {
	defer p.wg.Done()
	batch := make([]userWriteRequest, 0, p.writeBatchSize)
	flushTimer := time.NewTimer(p.writeFlushFreq)
	defer flushTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// RL.Logf("Flushing %d users", len(batch))
		for _, r := range batch {
			_ = r.ctx // currently unused, but could use store.WithContext
			if err := store.SaveUser(context.Background(), r.user); err != nil {
				// RL.Logf("history: save user error: %v", err)
				log.Printf("history: save envelope error:\n %v", err)
				r.result <- err
			} else {
				// RL.Logf("Saved user: %v", r.user.PeerID)
				r.result <- nil
			}
			close(r.result)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-p.stopCh:
			for {
				select {
				case req := <-p.writeQ:
					batch = append(batch, req)
					if len(batch) >= p.writeBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case req := <-p.writeQ:
			// RL.Logf("Dequeued user: %v", req.user.PeerID)
			batch = append(batch, req)
			if len(batch) >= p.writeBatchSize {
				flush()
				if !flushTimer.Stop() {
					<-flushTimer.C
				}
				flushTimer.Reset(p.writeFlushFreq)
			}
		case <-flushTimer.C:
			flush()
			flushTimer.Reset(p.writeFlushFreq)
		}
	}
}
