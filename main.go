package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("group-chat")

func getBootstraps() []string {
	return []string{
		"/ip4/52.62.79.81/tcp/33477/p2p/QmeKXgbrZhsymfSnS6NEHd24WAwE9DLEEPoVZJmD4LSset",
		"/ip4/13.211.33.184/tcp/33477/p2p/QmckF9nzR5152HtZveboBvEsZRqNJY6Dkr5vKpmmAjYRKe",
		"/ip4/54.253.132.182/tcp/33477/p2p/QmbaZ4JPREUb7WxeU4CrbZDs6ZMBTLb1Yc1o4fYGjJvxPf",
		"/ip4/3.25.154.110/tcp/33477/p2p/QmfEG3kx1rTa3oLEQrFMwT8D3kmPcfmRxJmRjtxChqGH9A",
	}
}

func main() {
	_ = log.SetLogLevel("group-chat", "info")
	// log.SetAllLoggers(log.LevelDebug)
	// _ = log.SetLogLevel("addrutil", "info")
	// _ = log.SetLogLevel("basichost", "info")

	log := messageLog{}
	log.data = make(map[message]struct{})

	ctx := context.Background()
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Routing(makeDhtRouting(ctx)),
		libp2p.EnableNATService(),
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		logger.Fatalf("could not start libp2p host: %v", err)
	}
	p, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		logger.Fatalf("could not start pubsub: %v", err)
	}
	t, err := p.Join("sylo-group-chat-demo")
	if err != nil {
		logger.Fatalf("could not join pubsub topic: %v", err)
	}
	sub, err := t.Subscribe()
	if err != nil {
		logger.Fatalf("could not subscribe to topic: %v", err)
	}
	go func() {
		for {
			next, err := sub.Next(ctx)
			if err != nil {
				logger.Fatalf("could not get next message: %v", err)
			}
			m := message{}
			err = json.Unmarshal(next.Data, &m)
			if err != nil {
				logger.Errorf("could not decode message: %v", err)
				continue
			}
			log.Append(m)
		}
	}()

	// var p2pReady sync.WaitGroup
	// p2pReady.Add(1)
	// go func() {
	// 	defer p2pReady.Done()
	// 	for {
	// 		time.Sleep(2 * time.Second)
	// 		for _, a := range h.Addrs() {
	// 			if strings.Contains(a.String(), "/p2p/") {
	// 				fmt.Printf("%v/p2p/%v\n", a, peer.Encode(h.ID()))
	// 				return
	// 			}
	// 		}
	// 	}
	// }()
	// p2pReady.Wait()

	fmt.Println("Shout into the void and see who shouts back...")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		// if strings.HasPrefix(s.Text(), "connect ") {
		// 	addr, err := multiaddr.NewMultiaddr(strings.TrimPrefix(s.Text(), "connect "))
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	info, err := peer.AddrInfoFromP2pAddr(addr)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	err = h.Connect(ctx, *info)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// }
		m := message{
			Clock: log.clock,
			ID:    peer.Encode(h.ID()),
			Text:  s.Text(),
		}
		b, err := json.Marshal(m)
		if err != nil {
			logger.Errorf("could not marshal message: %v", err)
			continue
		}
		err = t.Publish(ctx, b)
		if err != nil {
			logger.Errorf("could not publish message: %v", err)
			continue
		}
	}
	if s.Err() != nil {
		logger.Fatalf("input scanner error: %v", err)
	}
}

type message struct {
	Clock uint
	ID    string
	Text  string
}

type messageLog struct {
	mu    sync.Mutex
	data  map[message]struct{}
	clock uint
}

func (log *messageLog) Append(m message) {
	log.mu.Lock()
	defer log.mu.Unlock()
	if _, ok := log.data[m]; ok {
		return // we already have this message
	}
	logger.Infof("%s:\t%s", m.ID[len(m.ID)-6:len(m.ID)], m.Text)
	if m.Clock >= log.clock {
		log.clock = m.Clock + 1
	}
}

func makeDhtRouting(ctx context.Context) config.RoutingC {
	return func(h host.Host) (routing.PeerRouting, error) {
		d, err := dht.New(ctx, h)
		if err != nil {
			return nil, err
		}
		for _, bootstrap := range getBootstraps() {
			addr, err := multiaddr.NewMultiaddr(bootstrap)
			if err != nil {
				logger.Errorf("could not convert string to addr: %v", err)
				continue
			}
			info, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				logger.Errorf("could not convert addr to p2p info: %v", err)
				continue
			}
			err = h.Connect(ctx, *info)
			if err != nil {
				logger.Errorf("could not connect to bootstrap node: %v", err)
				continue
			}
		}
		return d, nil
	}
}
