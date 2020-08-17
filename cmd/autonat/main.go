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
		panic(err)
	}
	p, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		logger.Fatalf("could not start pubsub: %v", err)
	}
	t, err := p.Join("sylo-group-chat-demo")
	if err != nil {
		logger.Fatalf("could not join pubsub topic: %v", err)
	}
	cancel, err := t.Relay()
	if err != nil {
		logger.Fatalf("could not subscribe to topic: %v", err)
	}
	defer cancel()
	fmt.Printf("please wait 15 minutes for relay service...")
	select {
	case <-ctx.Done():
		return
	case <-time.After(15 * time.Minute):
	}
	fmt.Printf("done!")
	fmt.Println("autonat service ready to go!")
	fmt.Printf("/ip4/%v/tcp/%v/p2p/%v\n", os.Args[1], port, h.ID())
	select {}
}
