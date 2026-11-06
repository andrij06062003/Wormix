// Command wormix-core is the Wormix 1.05.0 game server: main server, PvP server, achievements server and
// (optionally) the Flash domain-policy server.
//
//	wormix-core <config.json>
//
// It runs until SIGINT/SIGTERM.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/andrij06062003/Wormix/services/core/internal/api"
	"github.com/andrij06062003/Wormix/services/core/internal/config"
	"github.com/andrij06062003/Wormix/services/core/internal/pvp"
	"github.com/andrij06062003/Wormix/services/core/internal/server"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetOutput(os.Stdout)
	if len(os.Args) < 2 {
		fmt.Printf("Use %s <config.json>\n", os.Args[0])
		os.Exit(2)
	}
	log.Printf("Wormix private server. Reading configuration %s", os.Args[1])
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Printf("can't read config: %v", err)
		os.Exit(1)
	}
	cfg, err := config.Parse(data)
	if err != nil {
		log.Printf("bad config: %v", err)
		os.Exit(1)
	}
	api.BaseURL = cfg.APIURL

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var servers []*server.Server
	seen := map[string]bool{}
	for _, name := range cfg.Order {
		sc, ok := cfg.Servers[name]
		if seen[name] || !ok || !sc.Enabled {
			continue
		}
		seen[name] = true
		s := server.New(name, net.JoinHostPort(cfg.Address(), strconv.Itoa(sc.Port)))
		if s == nil {
			continue
		}
		log.Printf("starting %s server at %s", name, s.Addr)
		if err := s.Start(); err != nil {
			log.Printf("can't start %s server: %v", name, err)
			for _, x := range servers {
				x.Close()
			}
			os.Exit(1)
		}
		servers = append(servers, s)
	}
	go pvp.RunTicker(ctx)
	log.Printf("setup completed, %d servers running", len(servers))

	<-ctx.Done()
	log.Printf("shutting down")
	for _, s := range servers {
		s.Close()
	}
}
