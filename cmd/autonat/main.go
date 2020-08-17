package main

import (
	"context"
	"flag"
	"strings"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("autonat")

func main() {
	log.SetAllLoggers(log.LevelDebug)
	_ = log.SetLogLevel("addrutil", "info")
	_ = log.SetLogLevel("basichost", "info")

	// check command line arguments
	a := flag.String("public-ip", "", "the public `IP` address of this autonat peer")
	flag.Parse()
	if a == nil || *a == "" {
		logger.Fatal("could not read autonat ip address")
	}

	ctx := context.Background()

	h, err := libp2p.New(ctx,
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) { return dht.New(ctx, h) }),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(relay.OptHop),
		libp2p.EnableAutoRelay(),
		libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			for _, addr := range addrs {
				if strings.Contains(addr.String(), "127.0.0.1") {
					p := strings.Replace(addr.String(), "127.0.0.1", *a, 1)
					ma, err := multiaddr.NewMultiaddr(p)
					if err != nil {
						continue
					}
					return []multiaddr.Multiaddr{ma}
				}
			}
			logger.Fatal("could not find any multiaddrs")
			return []multiaddr.Multiaddr{}
		}),
	)
	if err != nil {
		logger.Fatalf("could not start libp2p host: %v", err)
	}

	logger.Infof("I am a service peer!")
	for _, addr := range h.Addrs() {
		if strings.Contains(addr.String(), *a) {
			logger.Infof("Connect to me at: %v", addr)
		}
	}
	select {}
}
