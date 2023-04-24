package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	// &cli.IntFlag{ // TODO: Probably remove this once done debugging
	// 	Name:  "role",
	// 	Usage: "0 for gateway, 1 for worker (helpful for debugging)",
	// 	Value: 1,
	// },
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

	gateway, err := waitPeers(c, n)

	if err != nil {
		return err
	}

	// TODO: remove when done debugging
	// if c.Int("role") == 0 {
	// 	return runGateway(c, n)
	// }

	// TODO: add back when done debugging
	if gateway == n.Vat.Host.ID() {
		return runGateway(c, n)
	}

	return runWorker(c, n, gateway)
}

func gatewayHandler(ch csp.Chan, c *cli.Context, n *server.Node, writer http.ResponseWriter, request *http.Request) {

	// Receive value from synchronous channel
	f, r := ch.Recv(context.Background())
	defer r()

	// Turn received value into pointer
	ptr, err := f.Ptr()
	if err != nil {
		panic(err)
	}

	// Recover capability from client
	a := Echo(ptr.Interface().Client())

	// Convert to raw bytes
	b, err := io.ReadAll(request.Body)
	if err != nil {
		panic(err)
	}

	// Call capability with <request body>
	future, release := a.Echo(context.Background(), func(ps Echo_echo_Params) error {
		ps.SetPayload(b)
		return nil
	})
	defer release()

	// Wait for repsonse
	res, err := future.Struct()
	if err != nil {
		panic(err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(res.Segment().Data()) // TODO: Check this output if we have a bug
}

func gatewayHandlerWrapper(ch csp.Chan, c *cli.Context, n *server.Node) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		gatewayHandler(ch, c, n, writer, request)
	}
}

// curl -X POST -H "Content-Type: application/json" -d '{"message": "Hello, world!"}' http://localhost:8080/echo
func runGateway(c *cli.Context, n *server.Node) error {
	fmt.Println("Running Gateway...")

	var ch = csp.NewChan(&csp.SyncChan{})
	n.Vat.Export(chanCap, chanProvider{ch}) // Sets location to "/lb/chan"

	// run server
	http.HandleFunc("/echo", gatewayHandlerWrapper(ch, c, n))
	return http.ListenAndServe(":8080", nil)
}

func runWorker(c *cli.Context, n *server.Node, g peer.ID) error {
	fmt.Println("Worker booting up...")
	time.Sleep(20 * time.Second)
	fmt.Println("Running worker...")

	// Establish connection with gateway (corresponding to channel capability that gateway exported earlier)
	conn, err := n.Vat.Connect(c.Context, peer.AddrInfo{ID: g}, casm.BasicCap{"lb/chan", "lb/chan/packed"})

	if err != nil {
		return err
	}
	defer conn.Close()

	// Recover channel capability from Gateway
	a := csp.Chan(conn.Bootstrap(c.Context))

	// Busy loop sending request handler capabilities
	counter := 0
	for counter < 10 && err == nil {

		// Create capability
		server := EchoServer{}
		client := capnp.Client(Echo_ServerToClient(server))

		// Turn capability into pointer + Send pointer through channel
		ptr := csp.Client(client)
		logger.Info("Worker sending capability to channel")
		err = a.Send(context.Background(), ptr)

		if err != nil {
			return err
		}
		logger.Info("Msg success")
		time.Sleep(time.Second)
		counter++
	}

	return err
}

func waitPeers(c *cli.Context, n *server.Node) (peer.ID, error) {
	ctx, cancel := context.WithTimeout(c.Context, time.Second*30)
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
			return peer.ID(""), err
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
	return ps[0], nil
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
