package ux

import (
	"hillside/internal/profile"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestGenerateProfile(t *testing.T) {
	username := "testuser"
	password := "testpassword"

	prof, err := profile.GenerateProfile(username, password)
	if err != nil {
		t.Fatal(err)
	}
	if prof == nil {
		t.Fatal("profile is nil")
	}

	if prof.Username != username {
		t.Errorf("Expected username %s, got %s", username, prof.Username)
	}
	if prof.PasswordSalt == nil {
		t.Error("PasswordSalt is empty")
	}
	if prof.PasswordHash == nil {
		t.Error("PasswordHash is empty")
	}

	if prof.PeerID == "" {
		t.Error("PeerID is empty")
	}

	_, err = peer.Decode(prof.PeerID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadProfile(t *testing.T) {
	username := "tst"
	password := "tstspassword"

	prof, err := profile.GenerateProfile(username, password)
	if err != nil {
		t.Fatal(err)
	}

	loadedProf, err := profile.LoadProfile(username, password, "")
	if err != nil {
    	t.Fatalf("Failed to load profile: %v\nStack trace: %+v", err, err)
	}

	if loadedProf.Username != prof.Username {
		t.Errorf("Expected username %s, got %s", prof.Username, loadedProf.Username)
	}

}