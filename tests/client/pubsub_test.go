package client

import (
	"context"
	"fmt"
	"hillside/internal/p2p"
	"hillside/internal/profile"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestPS_Connection(t *testing.T) {
    ctx := context.Background()
    
    // Start Alice
    _, err := profile.GenerateProfile("Alice", "Malice")
    if err != nil {
        t.Fatal(err)
    }
    kbAlice, err := profile.LoadProfile("Alice", "Malice", "")
    if err != nil {
        t.Fatal(err)
    }
    
    nodeAlice := &p2p.Node{KB: kbAlice, Ctx: ctx}
    err = nodeAlice.InitHost([]string{"/ip4/0.0.0.0/tcp/0"})
    if err != nil {
        t.Fatalf("Failed to initialize Alice host: %v", err)
    }
    
    err = nodeAlice.InitDHT()
    if err != nil {
        t.Fatalf("Failed to initialize Alice DHT: %v", err)
    }
    
    err = nodeAlice.InitPubSub()
    if err != nil {
        t.Fatalf("Failed to initialize Alice PubSub: %v", err)
    }
    
    // Get Alice's address
    aliceAddr := fmt.Sprintf("%s/p2p/%s", nodeAlice.Host.Addrs()[0], nodeAlice.Host.ID().String())
    fmt.Printf("Alice listening on: %s\n", aliceAddr)
    
    // Alice subscribes to topic
    topicAlice, err := nodeAlice.PS.Join("test-topic")
    if err != nil {
        t.Fatalf("Alice failed to join topic: %v", err)
    }
    
    subAlice, err := topicAlice.Subscribe()
    if err != nil {
        t.Fatalf("Alice failed to subscribe: %v", err)
    }
    
    
    go func() {
        for {
            msg, err := subAlice.Next(ctx)
            if err != nil {
                return
            }
            message := string(msg.Data)
            fmt.Printf("Alice received: %s from %s\n", message, msg.ReceivedFrom)
        }
    }()
    
    time.Sleep(2 * time.Second)
    _, err = profile.GenerateProfile("Bob", "Balice")
    if err != nil {
        t.Fatal(err)
    }
    kbBob, err := profile.LoadProfile("Bob", "Balice", "")
    if err != nil {
        t.Fatal(err)
    }
    
    nodeBob := &p2p.Node{KB: kbBob, Ctx: ctx}
    err = nodeBob.InitHost([]string{"/ip4/0.0.0.0/tcp/0"})
    if err != nil {
        t.Fatalf("Failed to initialize Bob host: %v", err)
    }
    
    err = nodeBob.InitDHT()
    if err != nil {
        t.Fatalf("Failed to initialize Bob DHT: %v", err)
    }
    
    ai, err := peer.AddrInfoFromString(aliceAddr)
    if err != nil {
        t.Fatalf("Failed to parse Alice's address: %v", err)
    }
    
    err = nodeBob.Host.Connect(ctx, *ai)
    if err != nil {
        t.Fatalf("Failed to connect Bob to Alice: %v", err)
    }
    fmt.Printf("Bob connected to Alice\n")
    
    err = nodeBob.InitPubSub()
    if err != nil {
        t.Fatalf("Failed to initialize Bob PubSub: %v", err)
    }
    
    time.Sleep(3 * time.Second)
    topicBob, err := nodeBob.PS.Join("test-topic")
    if err != nil {
        t.Fatalf("Bob failed to join topic: %v", err)
    }
    
    message := "Hello from Bob!"
    err = topicBob.Publish(ctx, []byte(message))
    if err != nil {
        t.Fatalf("Failed to publish message: %v", err)
    }
    fmt.Printf("Bob published: %s\n", message)
    time.Sleep(5 * time.Second)
}