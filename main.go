package main

import (
	"context"
	"errors"
	"os"
	"sort"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"
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
	&cli.IntFlag{
		Name:  "num-peers",
		Usage: "number of expected peers in the cluster",
		Value: 2,
	},
}

var (
	logger log.Logger
	n      *server.Node
)

func main() {
	app := &cli.App{
		Action: run,
		Flags:  flags,
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// TODO: what do we add to have something to constantly handle requests from clients?
	app := fx.New(runtime.NewServer(c.Context, c),
		fx.Populate(&logger, &n),
		fx.Supply(c))

	if err := app.Start(c.Context); err != nil {
		return err
	}
	defer app.Stop(context.Background())

	logger.Info("server started")

	// wait until all peers have joined the cluster, so that we
	// can check to see who the gateway is.
	gateway, err := waitPeers(c, n)
	if err != nil {
		return err
	}

	// are we the gateway peer?
	if gateway {
		return runGateway(c, n)
	}

	// if not, then we're a worker.
	return runWorker(c, n)
}

func runGateway(c *cli.Context, n *server.Node) error {
	var ch csp.Chan                         // TODO:  csp.NewChan()
	n.Vat.Export(chanCap, chanProvider{ch}) // TODO: Will this also be exported to peers we find in the future?

	return errors.New("NOT IMPLEMENTED")
}

func runWorker(c *cli.Context, n *server.Node) error {
	return errors.New("NOT IMPLEMENTED")
}

func waitPeers(c *cli.Context, n *server.Node) (bool, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*10)
	defer cancel()

	ps := make(peerSlice, 0, c.Int("num-peers"))

	log := logger.With(n).
		WithField("n_peers", cap(ps))
	log.Info("waiting for peers")

	for len(ps) < cap(ps) {
		it, release := n.View().Iter(ctx, queryAll())
		defer release()

		for r := it.Next(); r != nil; r = it.Next() {
			ps = append(ps, r.Peer())
		}

		if err := it.Err(); err != nil {
			return false, err
		}

		// did we find everyone?
		if len(ps) < cap(ps) {
			logger.Infof("found %d peers", len(ps))
			release()
			ps = ps[:0] // reset length to 0
			time.Sleep(time.Millisecond * 100)
			continue
		}

		logger.With(n).Info("found all peers")
		break
	}

	sort.Sort(ps)
	return n.Vat.Host.ID() == ps[0], nil
}

type peerSlice []peer.ID

func (ps peerSlice) Len() int           { return len(ps) }
func (ps peerSlice) Less(i, j int) bool { return ps[i] < ps[j] }
func (ps peerSlice) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

var chanCap = casm.BasicCap{"/lb/chan"} // Set the location of the channel

type chanProvider struct{ csp.Chan }

func (cp chanProvider) Client() capnp.Client {
	return capnp.Client(cp.Chan)
}

func queryAll() cluster.Query {
	return cluster.NewQuery(cluster.All())
}

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
