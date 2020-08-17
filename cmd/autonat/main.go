package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var logger = log.Logger("autonat")

const port = 33477

func main() {
	ctx := context.Background()
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%v", port)),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) { return dht.New(ctx, h) }),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(relay.OptHop),
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		logger.Fatalf("could not start host: %v", err)
	}
	fmt.Printf("please wait for relay service to begin...")
	select {
	case <-ctx.Done():
		return
	case <-time.After(16 * time.Minute):
	}
	p, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		logger.Fatalf("could not start pubsub: %v", err)
	}
	t, err := p.Join("sylo-group-chat-demo")
	if err != nil {
		logger.Fatalf("could not join pubsub topic: %v", err)
	}
	_, err = t.Subscribe()
	if err != nil {
		logger.Fatalf("could not subscribe to topic: %v", err)
	}
	fmt.Printf("done!\n")
	fmt.Println("autonat service ready to go!")
	fmt.Printf("/ip4/%v/tcp/%v/p2p/%v\n", os.Args[1], port, h.ID())
	select {}
}
