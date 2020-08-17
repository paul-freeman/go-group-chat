package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

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
	fmt.Println("I am an autonat service and I am ready to go!")
	fmt.Printf("/ip4/%v/tcp/%v/p2p/%v\n", os.Args[1], port, h.ID())
	select {}
}
