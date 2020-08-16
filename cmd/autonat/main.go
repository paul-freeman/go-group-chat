package main

import (
	"context"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

var logger = log.Logger("autonat")

func main() {
	log.SetAllLoggers(log.LevelDebug)
	_ = log.SetLogLevel("addrutil", "info")
	_ = log.SetLogLevel("basichost", "info")

	ctx := context.Background()

	h, err := libp2p.New(ctx,
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) { return dht.New(ctx, h) }),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(relay.OptHop),
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		logger.Fatalf("could not start libp2p host: %v", err)
	}

	logger.Infof("I am a service peer!")
	logger.Infof("Connect to me at:")
	for i, a := range h.Addrs() {
		logger.Infof("%d\t%v/p2p/%v", i+1, a, h.ID())
	}
	select {}
}
