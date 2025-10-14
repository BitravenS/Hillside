package p2p_test

import (
	"hillside/internal/p2p"
	"testing"
)

func TestInitHistoryManager(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		dbPath     string
		writeQSize int
		wantErr    bool
	}{
		{
			name:       "Valid parameters",
			username:   "testuser",
			dbPath:     "",
			writeQSize: 100,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := p2p.InitHistoryManager(tt.username, tt.dbPath, tt.writeQSize)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("InitHistoryManager() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("InitHistoryManager() succeeded unexpectedly")
			}
			if true {
				t.Errorf("InitHistoryManager() = %v", got)
			}
		})
	}
}
