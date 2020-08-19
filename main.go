package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("group-chat")

const topic = "sylo-group-chat-demo"

type message struct {
	Clock uint
	ID    string
	Name  string
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
		// we already have this message
		return
	}
	name := m.Name
	if name == "" {
		name = m.ID[len(m.ID)-6 : len(m.ID)]
	}
	logger.Infof("%s:\t%s", name, m.Text)
	if m.Clock >= log.clock {
		log.clock = m.Clock + 1
	}
}

type bootstraps []*peer.AddrInfo

func (bs *bootstraps) String() string {
	strs := make([]string, len(*bs))
	for i, addr := range *bs {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (bs *bootstraps) Set(str string) error {
	addr, err := multiaddr.NewMultiaddr(str)
	if err != nil {
		return err
	}
	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return err
	}
	*bs = append(*bs, info)
	return nil
}

func main() {
	_ = log.SetLogLevel("group-chat", "info")

	// parse command line arguments
	var bs bootstraps
	var name string
	var hop bool
	var ro bool
	var ip string

	flag.Var(&bs, "bootstrap", "will connect to this `PEER` to bootstrap the network")
	flag.StringVar(&name, "nickname", "", "this `NAME` will be attached to your messages")
	flag.BoolVar(&hop, "relay", false, "allows other peers to relay through this peer")
	flag.BoolVar(&ro, "read-only", false, "disable input and just observe the chat")
	flag.StringVar(&ip, "ip", "", "public `IP` address (for relay peers)")
	flag.Parse()

	if hop && ip == "" {
		logger.Fatal("a public ip address is required when starting as a relay")
	}

	// create message log
	log := messageLog{}
	log.data = make(map[message]struct{})

	// start libp2p host
	ctx := context.Background()
	enableRelay := libp2p.EnableRelay()
	if hop {
		enableRelay = libp2p.EnableRelay(relay.OptHop)
	}
	h, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) { return dht.New(ctx, h) }),
		libp2p.EnableNATService(),
		enableRelay,
		libp2p.EnableAutoRelay(),
	)
	if err != nil {
		logger.Fatalf("could not start libp2p host: %v", err)
	}

	// subscribe to messages
	p, err := pubsub.NewFloodSub(ctx, h)
	if err != nil {
		logger.Fatalf("could not start pubsub: %v", err)
	}
	t, err := p.Join(topic)
	if err != nil {
		logger.Fatalf("could not join pubsub topic: %v", err)
	}
	sub, err := t.Subscribe()
	if err != nil {
		logger.Fatalf("could not subscribe to topic: %v", err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
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

	// connect to bootstrap peers
	for _, b := range bs {
		if err := h.Connect(ctx, *b); err != nil {
			logger.Errorf("could not connect to bootstrap node: %v", err)
			continue
		}
	}
	if hop {
		fmt.Println("relay peers must wait 15 minutes before use")
		time.Sleep(15 * time.Minute)
		fmt.Println("ready to go")
		for _, a := range h.Addrs() {
			if strings.Contains(a.String(), "127.0.0.1") {
				fmt.Printf("%v/p2p/%v\n", strings.Replace(a.String(), "127.0.0.1", ip, 1), h.ID())
			}
		}
	}
	if ro {
		// read only
		<-ctx.Done()
		return
	}

	// send messages
	fmt.Println("welcome to the chat!")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		m := message{
			Clock: log.clock,
			ID:    peer.Encode(h.ID()),
			Name:  name,
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
