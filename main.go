package main

import (
	"context"
	"flag"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/config"
	ma "github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("group-chat")

func main() {
	log.SetAllLoggers(log.LevelDebug)
	_ = log.SetLogLevel("addrutil", "info")

	// check command line arguments
	a := flag.String("autonat", "", "the P2P multiaddr of the autonat peer")
	flag.Parse()
	if a == nil || *a == "" {
		logger.Fatal("could not read autonat address")
	}
	mAddr, err := ma.NewMultiaddr(*a)
	if err != nil {
		logger.Fatalf("could not convert %s to a multiaddr: %v", *a, err)
	}
	p2pAddr, err := peer.AddrInfoFromP2pAddr(mAddr)
	if err != nil {
		logger.Fatalf("could not convert %s to a P2P address: %v", *a, err)
	}

	// start the application context
	ctx := context.Background()

	// start libp2p host
	h, err := libp2p.New(ctx,
		libp2p.Routing(makeDhtRouting(ctx, *p2pAddr)),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		logger.Fatalf("could not start libp2p host: %v", err)
	}

	err = h.Connect(ctx, *p2pAddr)
	if err != nil {
		panic(err)
	}

	// monitor our multiaddrs
	for {
		for i, a := range h.Addrs() {
			if i == 0 {
				logger.Infof("We have the following multiaddrs:")
			}
			logger.Infof("%d:\t%v", i, a)
		}
		time.Sleep(5 * time.Second)
	}
}

func makeDhtRouting(ctx context.Context, bootstrap peer.AddrInfo) config.RoutingC {
	return func(h host.Host) (routing.PeerRouting, error) {
		d, err := dht.New(ctx, h)
		if err != nil {
			return nil, err
		}
		err = h.Connect(ctx, bootstrap)
		if err != nil {
			return nil, err
		}
		return d, nil

	}
}
