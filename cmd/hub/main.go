package hub

import (
	"context"
	"hillside/internal/hub"
	"log"
)

func main() {
	ctx := context.Background()
	h, err := hub.NewHubServer(ctx, "/ip4/0.0.0.0/tcp/4001")
	if err != nil {
		log.Fatal(err)
	}
	h.ListenAddrs()
	select {}

}