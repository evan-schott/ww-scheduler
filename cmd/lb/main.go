package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
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

	lb "github.com/evan-schott/ww-load-balancer"
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
	&cli.BoolFlag{
		Name:  "gateway",
		Usage: "number of expected peers in the cluster",
	},
	&cli.StringFlag{
		Name:  "dial",
		Usage: "id of gateway peer",
	},
}

var (
	logger log.Logger
	n      *server.Node
)

func main() {
	app := createApp()

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}

func createApp() *cli.App {
	app := &cli.App{
		Action: run,
		Flags:  flags,
	}
	return app
}

func run(c *cli.Context) error {
	app := fx.New(runtime.NewServer(c.Context, c),
		fx.Populate(&logger, &n),
		fx.Supply(c))

	if err := app.Start(c.Context); err != nil {
		return err
	}
	defer app.Stop(context.Background())

	logger.Info("server started")

	if c.Bool("gateway") {
		return runGateway(c, n)
	}

	return runWorker(c, n)
}

func runGateway(c *cli.Context, n *server.Node) error {
	log := logger.With(n)
	log.Info("gateway started")

	ch := csp.NewChan(&csp.SyncChan{})
	defer ch.Close(c.Context)

	n.Vat.Export(chanCap, chanProvider{ch})

	log.Info("exported channel")

	return http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// each HTTP request is served in its own goroutine

		// Get the request payload
		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Block on <-ch until either we get a result, or the request
		// context expires.  In the latter case, this indicates that
		// the client aborted.
		f, release := ch.Recv(r.Context())
		defer release()

		// Get the worker's echo capability
		echo := lb.Echo(f.Client())

		// Call the handler's RPC method
		e, release := echo.Echo(r.Context(), lb.Data(b))
		defer release()

		// block until we get the RPC response
		res, err := e.Struct()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// Grab the echoed bytes from the response
		p, err := res.Response()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// Send the response back to the caller
		_, err = io.Copy(w, bytes.NewReader(p))
		if err != nil {
			log.WithError(err).
				Warn("failed to write echo response")
		}
	}))
}

func runWorker(c *cli.Context, n *server.Node) error {
	if err := waitGateway(c, n); err != nil {
		return err
	}

	// Routing information for the gateway server.
	gateway, err := peer.Decode(c.String("dial"))
	if err != nil {
		return err
	}

	addr := peer.AddrInfo{ID: gateway}

	log := logger.With(&addr)
	log.Info("worker started")

	// Establish connection with gateway (corresponding to channel capability that gateway exported earlier)
	conn, err := n.Vat.Connect(c.Context, addr, chanCap)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Recover channel capability from Gateway.  Cast it as a send-
	// only channel, to avoid mistakes.
	ch := csp.Sender(conn.Bootstrap(c.Context))

	// Loop until the worker starts shutting down.
	for c.Context.Done() == nil {
		// Setup of the echo capability. Start an echo server, and derive a client from it.
		server := lb.EchoServer{}
		e := capnp.Client(lb.Echo_ServerToClient(server))

		// Block until we're able to send our echo capability to the
		// gateway server.  This is where the load-balancing happens.
		// We are competing with other Send()s, and the gateway will
		// pick one of the senders at random each time it handles an
		// HTTP request.
		err := ch.Send(context.Background(), csp.Client(e))
		if err != nil {
			return err // this generally means the gateway is down
		}
	}

	return nil
}

func waitGateway(c *cli.Context, n *server.Node) error {
	for {
		ok, err := queryForGateway(c, n)
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		time.Sleep(time.Millisecond * 500)
	}
}

func queryForGateway(c *cli.Context, n *server.Node) (bool, error) {
	it, release := n.View().Iter(c.Context, matchAll())
	defer release()

	for r := it.Next(); r != nil; r = it.Next() {
		if r != nil && r.Peer().String() == c.String("dial") {
			return true, nil
		}
	}

	return false, it.Err()
}

var chanCap = casm.BasicCap{"/lb"} // Set the location of the channel

type chanProvider struct{ csp.Chan }

func (cp chanProvider) Client() capnp.Client {
	return capnp.Client(cp.Chan)
}

func matchAll() cluster.Query {
	return cluster.NewQuery(cluster.All())
}
