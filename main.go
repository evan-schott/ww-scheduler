package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	csp "github.com/wetware/ww/pkg/csp"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/server"
	"go.uber.org/fx"
)

// Used for cli.App()
var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic",
			"/ip6/::0/udp/0/quic"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringSliceFlag{
		Name:    "addr",
		Aliases: []string{"a"},
		Usage:   "static bootstrap `ADDR`",
		EnvVars: []string{"WW_ADDR"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "bootstrap discovery multiaddr",
		Value:   "/ip4/228.8.8.8/udp/8822/multicast/lo0", // VPN:
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringSliceFlag{
		Name:    "meta",
		Usage:   "metadata fields in key=value format",
		EnvVars: []string{"WW_META"},
	},
	&cli.StringFlag{
		Name:  "gateway",
		Usage: "multiaddr of gateway peer",
	},
}

func main() {
	app := &cli.App{
		Action: run,
		Flags:  flags,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// TODO: what do we add to have something to constantly handle requests from clients?
	app := fx.New(runtime.NewServer(c.Context, c),
		fx.Supply(c),
		fx.Invoke(bind))

	if err := app.Start(c.Context); err != nil {
		return err
	}

	log.Info("we're up!")
	<-app.Done()

	return app.Stop(context.Background())
}

func bind(c *cli.Context, vat casm.Vat, n *server.Node) error {
	if c.IsSet("gateway") {
		// it's a worker peer - get its address, connnect and grab the channel
	} else {
		// it's the gateway peer
	}

	var ch csp.Chan
	vat.Export(chanCap, chanProvider{ch}) // TODO: Will this also be exported to peers we find in the future?

	time.Sleep(10 * time.Second) // Give time for everyone else to start up

	view := n.Cluster.View()
	it, release := view.Iter(context.Background(), selectAll())
	defer release()
	fmt.Println("woke up")
	fmt.Println("my id:", vat.Host.ID())

	min := vat.Host.ID()                                // If we take just last byte then isn't there big chance we tie?
	for rec := it.Next(); rec != nil; rec = it.Next() { // TODO: Couldn't find any other peers when ran this before

		fmt.Println("Found peer:", rec.Peer())
		if rec.Peer() < min {
			min = rec.Peer()
		}
	}
	if it.Err() != nil {
		return it.Err()
	}
	fmt.Println("lowest peer is:", min)

	if min == vat.Host.ID() { // Case we are gateway
		// now we gotta actually set up a channel on our end
		// csp.NewChan() // Confused why peekable chan is loading but not NewChan(), something to do with how set up go

	}
	// else { // Case we are worker
	// 	server := echo.EchoServer{}
	// 	client := echo.Echo_ServerToClient(server)

	// 	// how access leader chan (given we know address for it is gatewayID/lb/chan)

	// 	// do we need all that rpc stuff or is there other way?
	// 	// Use vat.Connect(ctx, min, )?

	// }

	// TODO: Case where we are the leader
	// send capability of channel?
	// wait for requests + respond with popping from channel

	// conn, err := vat.Connect(context.Background(), addr, chanCap)
	// if err != nil {
	// 	return err
	// }

	// client := conn.Bootstrap(context.Background())
	// ch := csp.Chan(client)

	return nil
}

var chanCap = casm.BasicCap{"/lb/chan"} // Set the location of the channel

type chanProvider struct{ csp.Chan }

func (cp chanProvider) Client() capnp.Client {
	return capnp.Client(cp.Chan)
}

func selectAll() cluster.Query {
	return cluster.NewQuery(cluster.Match(matchAll{}))
}

type matchAll struct{}

func (matchAll) String() string { return "host" }
func (matchAll) Prefix() bool   { return true }
func (matchAll) Host() string   { return "/" }

// casm ~ vat (layer of abstraction on top of libp2p) => so capnp capabilities on libp2p (vs raw libp2p streams)

// TODO LIST:
// 1. have peers determine gateway (impl: tail end of peerID random, take last byte to determine),
// 2. make gateway start http server
// 3. setup endpoint that receives payload and echos it back
// 4. make sure can work with curl
// 5. make gateway peer create/export channel
// 6. all workers connect and get channel
// 7. make all workers create echo server (don't export)
// [8]. have workers in infinite loop try to send capability to channel
// [9]. make gateway http handler for each incoming request recv from channel (echo cap), call echo, return result (after handled by worker)

// turn capability into pointer
