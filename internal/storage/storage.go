// Package storage implements the SQLite storage backend for history, auth and peer management.
package storage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

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
		return nil, ErrCannotConnect.WithDetails("Base directory does not exist")
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
	if err != nil {
		return err
	}
	err = s.MigrateAuth()
	return err
}
