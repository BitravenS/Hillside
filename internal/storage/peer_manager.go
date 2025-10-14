package storage

import (
	"context"
	"errors"
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
