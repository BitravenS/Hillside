package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"hillside/internal/models"
	"hillside/internal/utils"
)

var ErrNoRows = errors.New("no rows")

type Store struct {
	db *sql.DB
}

type SessionDB struct {
	Store   *Store
	History *HistoryManager
	Peers   *PeerManager
}

// NewSQLiteStore opens (or creates) a sqlite DB file.
// dsn example: "file:history.db?_foreign_keys=1" — see defaults below.
func NewSQLiteStore(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set synchronous: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign_keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	return &Store{db: db}, nil
}

func InitSessionDB(username string, dbPath string, writeQSize int) (*SessionDB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filldbPath := homeDir + "/.hillside/" + fmt.Sprintf("hillside_data_%s.db", username)

	if dbPath != "" {
		filldbPath = dbPath
	}
	_, err = os.Stat(homeDir + "/.hillside/")
	if os.IsNotExist(err) {
		return nil, utils.HistoryDBNotFound
	}
	store, err := NewSQLiteStore(filldbPath)
	if err != nil {
		return nil, fmt.Errorf("init  %w", err)
	}
	if err := store.Migrate(); err != nil {
		store.Close()
		return nil, err
	}
	h := NewHistoryManager(writeQSize)
	h.Start(store)

	p := NewPeerManager(writeQSize)
	p.Start(store)

	sdb := &SessionDB{
		History: h,
		Store:   store,
		Peers:   p,
	}
	return sdb, nil
}

func (s *Store) Close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

// Migrate creates the messages table and necessary indexes.
// This is idempotent.
func (s *Store) Migrate() error {
	const sqlStmt = `
CREATE TABLE IF NOT EXISTS messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  room_id TEXT NOT NULL,
  server_id TEXT,
  chain_index INTEGER, -- nullable
  msg_type TEXT NOT NULL,
  sender_id TEXT,
  timestamp INTEGER NOT NULL, -- unix micro
  signature BLOB NOT NULL,
	payload BLOB NOT NULL
);

-- Unique index for chat messages (chain_index not null)
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_chain ON messages (room_id, chain_index) WHERE chain_index IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_room_time ON messages (room_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_room_chain ON messages (room_id, chain_index DESC);
CREATE INDEX IF NOT EXISTS idx_sender ON messages (sender_id);

-- Create peer table for storing peer info

CREATE TABLE IF NOT EXISTS peers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		peer_id TEXT NOT NULL UNIQUE,
		dilithium_pub BLOB NOT NULL,
		kyber_pub BLOB NOT NULL,
		libp2p_pub BLOB NOT NULL,
		username TEXT,
		color TEXT,
		last_seen INTEGER, -- unix micro
		synced INTEGER DEFAULT 1 -- boolean (0/1)
);

-- Unique index for peer_id
CREATE UNIQUE INDEX IF NOT EXISTS uq_peer_id ON peers (peer_id);
CREATE INDEX IF NOT EXISTS idx_username_id ON peers (peer_id, username);


`
	_, err := s.db.Exec(sqlStmt)
	return err
}

// SaveEnvelope stores an envelope. Use chainIndex != nil for chat messages.
// Behavior: insert is idempotent (duplicate chain_index for same room ignored).
func (s *Store) SaveEnvelope(ctx context.Context, raw []byte, env *models.Envelope, chainIndex *uint64, roomID string, serverID string) error {
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
		string(env.Type),
		env.Sender.PeerID,
		env.Timestamp,
		env.Signature,
		env.Payload,
	)
	if err != nil {
		return fmt.Errorf("insert envelope: %w", err)
	}
	return nil
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

// GetMessagesSinceChainIndex returns chat messages with chain_index > sinceIndex ordered ASC.
func (s *Store) GetMessagesSinceChainIndex(ctx context.Context, roomID string, sinceIndex uint64, limit int) ([]models.StoredMessage, error) {
	const q = `
SELECT id, room_id, server_id, chain_index, msg_type, sender_id, timestamp, signature, payload
FROM messages
WHERE room_id = ? AND chain_index IS NOT NULL AND chain_index > ?
ORDER BY chain_index ASC
LIMIT ?;
`
	rows, err := s.db.QueryContext(ctx, q, roomID, int64(sinceIndex), limit)
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
  chain_index DESC,
  timestamp DESC
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
