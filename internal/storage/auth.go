package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type RoomAuth struct {
	RoomID           string
	ChainIndex       uint64
	MasterRatchetKey []byte    // current master ratchet key (32 bytes)
	LastUsed         time.Time // last time this ratchet was updated/used (UnixMicro stored)
	Tombstone        bool      // soft-delete flag
	Synced           bool      // whether this row is synced to remote (0/1)
}

// MigrateAuth creates the room_auth table and useful indexes.
// Call this once during store migration/initialization (or include it in Store.Migrate).
func (s *Store) MigrateAuth() error {
	const sqlStmt = `
CREATE TABLE IF NOT EXISTS room_auth (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	room_id TEXT NOT NULL UNIQUE,
	chain_index INTEGER DEFAULT 0, -- last used chain index
	master_ratchet_key BLOB, -- current master ratchet key (32 bytes)
	last_used INTEGER NOT NULL, -- unix micro
	tombstone INTEGER DEFAULT 0, -- 0/1
	synced INTEGER DEFAULT 1 -- 0/1
);

CREATE INDEX IF NOT EXISTS idx_room_auth_last_used ON room_auth (last_used DESC);
CREATE INDEX IF NOT EXISTS idx_room_auth_tombstone ON room_auth (tombstone);
`
	_, err := s.db.Exec(sqlStmt)
	return err
}

func (s *Store) SaveAuth(ctx context.Context, roomID string, chainIdx int64, masterKey []byte, lastUsed time.Time) error {
	lu := lastUsed.UnixMicro()
	const q = `
INSERT INTO room_auth (room_id, chain_index, master_ratchet_key, last_used, tombstone, synced)
VALUES (?, ?, ?, ?, 0, 1)
ON CONFLICT(room_id) DO UPDATE SET
		chain_index = excluded.chain_index,
		master_ratchet_key = excluded.master_ratchet_key,
    last_used = excluded.last_used,
    tombstone = 0,
    synced = 1;
`

	_, err := s.db.ExecContext(ctx, q, roomID, chainIdx, masterKey, lu)
	if err != nil {
		return fmt.Errorf("save auth (auto inc): %w", err)
	}
	return nil
}

// GetAuth returns the stored blob and metadata for a room. ErrNoRows if not found.
func (s *Store) GetAuth(ctx context.Context, roomID string) (*RoomAuth, error) {
	const q = `
SELECT room_id, chain_index, master_ratchet_key, last_used, tombstone, synced
FROM room_auth
WHERE room_id = ?
LIMIT 1;
`
	row := s.db.QueryRowContext(ctx, q, roomID)
	var (
		rid       string
		chainIdx  sql.NullInt64
		masterKey []byte
		lastUsed  sql.NullInt64
		tombstone sql.NullInt64
		synced    sql.NullInt64
	)
	if err := row.Scan(&rid, &chainIdx, &masterKey, &lastUsed, &tombstone, &synced); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRows
		}
		return nil, fmt.Errorf("get auth scan: %w", err)
	}
	if !lastUsed.Valid {
		return nil, fmt.Errorf("get auth: invalid last_used")
	}
	ra := &RoomAuth{
		RoomID:           rid,
		ChainIndex:       uint64(chainIdx.Int64),
		MasterRatchetKey: masterKey,
		LastUsed:         time.UnixMicro(lastUsed.Int64),
		Tombstone:        tombstone.Valid && tombstone.Int64 != 0,
		Synced:           !synced.Valid || synced.Int64 != 0,
	}

	return ra, nil
}

// DeleteAuth hard-deletes the auth row for a room.
func (s *Store) DeleteAuth(ctx context.Context, roomID string) error {
	const q = `DELETE FROM room_auth WHERE room_id = ?;`
	_, err := s.db.ExecContext(ctx, q, roomID)
	if err != nil {
		return fmt.Errorf("delete auth: %w", err)
	}
	return nil
}

// SoftDeleteAuth marks the row tombstoned (keeps historical data for potential recovery/sync).
func (s *Store) SoftDeleteAuth(ctx context.Context, roomID string) error {
	const q = `UPDATE room_auth SET tombstone = 1, synced = 0 WHERE room_id = ?;`
	_, err := s.db.ExecContext(ctx, q, roomID)
	if err != nil {
		return fmt.Errorf("soft delete auth: %w", err)
	}
	return nil
}

// UpdateLastUsed sets last_used for the given room (fast path).
func (s *Store) UpdateLastUsed(ctx context.Context, roomID string, lastUsed time.Time) error {
	const q = `UPDATE room_auth SET last_used = ? WHERE room_id = ?;`
	_, err := s.db.ExecContext(ctx, q, lastUsed.UnixMicro(), roomID)
	if err != nil {
		return fmt.Errorf("update last_used: %w", err)
	}
	return nil
}

// ListAuths returns all auth entries (optionally include tombstones). Caller can limit/offset if needed.
func (s *Store) ListAuths(ctx context.Context, includeTombstones bool) ([]*RoomAuth, error) {
	const qBase = `
SELECT room_id, chain_index, master_ratchet_key, last_used, tombstone, synced
FROM room_auth
`
	var q string
	if includeTombstones {
		q = qBase + " ORDER BY last_used DESC;"
	} else {
		q = qBase + " WHERE tombstone = 0 ORDER BY last_used DESC;"
	}
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list auths: %w", err)
	}
	defer rows.Close()
	out := make([]*RoomAuth, 0)
	for rows.Next() {
		var (
			rid       string
			chainIdx  sql.NullInt64
			masterKey []byte
			lastUsed  sql.NullInt64
			tombstone sql.NullInt64
			synced    sql.NullInt64
		)
		if err := rows.Scan(&rid, &chainIdx, &masterKey, &lastUsed, &tombstone, &synced); err != nil {
			return nil, fmt.Errorf("list auths scan: %w", err)
		}
		ra := &RoomAuth{
			RoomID:           rid,
			ChainIndex:       uint64(chainIdx.Int64),
			MasterRatchetKey: masterKey,
			LastUsed:         time.UnixMicro(lastUsed.Int64),
			Tombstone:        tombstone.Valid && tombstone.Int64 != 0,
			Synced:           !synced.Valid || synced.Int64 != 0,
		}

		out = append(out, ra)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// PurgeAuthsOlderThan deletes auth entries with last_used < before.
// Returns number of rows deleted.
func (s *Store) PurgeAuthsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	const q = `DELETE FROM room_auth WHERE last_used < ? AND tombstone = 1;`
	res, err := s.db.ExecContext(ctx, q, before.UnixMicro())
	if err != nil {
		return 0, fmt.Errorf("purge auths older than: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
