package client

import (
	"context"
	"fmt"
	"hillside/internal/p2p"
	"hillside/internal/profile"
	"testing"
)

func TestClientNodeInitialization(t *testing.T) {
	username := "tst"
	password := "tstspassword"

	// Generate a profile first
	_, err := profile.GenerateProfile(username, password)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := profile.LoadProfile(username, password, "")
	if err != nil {
    	t.Fatalf("Failed to load profile: %v\nStack trace: %+v", err, err)
	}
	ctx := context.Background()
	node := &p2p.Node{
		KB: kb,
		Ctx: ctx,}
	err = node.InitHost([]string{"/ip4/0.0.0.0/tcp/0"})
	if err != nil {
		t.Fatalf("Failed to initialize host: %v", err)
	}
	err = node.InitDHT()
	if err != nil {
		t.Fatalf("Failed to initialize DHT: %v", err)
	}
	err = node.InitPubSub()
	if err != nil {
		t.Fatalf("Failed to initialize PubSub: %v", err)
	}
	fmt.Printf("Node initialized with PeerID: %s\n", node.Host.ID().String())
	for _, addr := range node.Host.Addrs() {
		fmt.Printf("Listening on: %s/p2p/%s\n", addr, node.Host.ID().String())
	}
}