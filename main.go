package main

import (
	"context"
	"flag"
	"strings"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/config"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("group-chat")

func main() {
	log.SetAllLoggers(log.LevelDebug)
	_ = log.SetLogLevel("addrutil", "info")
	_ = log.SetLogLevel("basichost", "info")

	// check command line arguments
	a := flag.String("autonat", "", "a comma-separated list of multiaddrs")
	flag.Parse()
	if a == nil || *a == "" {
		logger.Fatal("could not read autonat address")
	}
	infos := []*peer.AddrInfo{}
	for _, addr := range strings.Split(*a, ",") {
		mAddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			logger.Errorf("could not convert %s to a multiaddr: %v", *a, err)
			continue
		}
		p2pAddr, err := peer.AddrInfoFromP2pAddr(mAddr)
		if err != nil {
			logger.Errorf("could not convert %s to a P2P address: %v", *a, err)
			continue
		}
		infos = append(infos, p2pAddr)
	}

	// start the application context
	ctx := context.Background()

	// start libp2p host
	h, err := libp2p.New(ctx,
		libp2p.Routing(makeDhtRouting(ctx, infos)),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		logger.Fatalf("could not start libp2p host: %v", err)
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

func makeDhtRouting(ctx context.Context, bootstraps []*peer.AddrInfo) config.RoutingC {
	return func(h host.Host) (routing.PeerRouting, error) {
		d, err := dht.New(ctx, h)
		if err != nil {
			return nil, err
		}
		for _, bootstrap := range bootstraps {
			err = h.Connect(ctx, *bootstrap)
			if err != nil {
				logger.Errorf("could not connect to bootstrap node: %v", err)
				continue
			}
		}
		return d, nil
	}
}
