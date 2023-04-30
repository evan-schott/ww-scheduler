package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"capnproto.org/go/capnp/v3"
	worker "github.com/evan-schott/ww-scheduler"
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
		//Value:   "/ip4/239.0.0.1/udp/12345/multicast/eth0",
		Value:   "/ip4/228.8.8.8/udp/8822/multicast/lo0", // TODO: change to loopback to run locally  "/ip4/228.8.8.8/udp/8822/multicast/lo0"
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

type Task struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completions int    `json:"complete"`
	Duration    int    `json:"duration"`
	Start       int    `json:"start"`
	Delay       int    `json:"delay"`
	Repeats     int    `json:"repeats"`
	Wasm        []byte `json:"wasm"`
}

var tasks = struct {
	sync.RWMutex
	m map[string]Task
}{m: make(map[string]Task)}

type WorkerTuple struct {
	id          int
	connections int
	cap         worker.Worker
}

type WorkerMap struct {
	mapping map[int]*WorkerTuple
	mu      sync.Mutex
}

var workerMap = &WorkerMap{
	mapping: make(map[int]*WorkerTuple),
}

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

	// Launch go routine to receive from channel
	go func() {
		log.Info("starting go routine to listen for new workers trying to join")

		// Loop until the gateway starts shutting down.
		for c.Context.Done() == nil {

			// Block on <-ch until we get a result
			f, release := ch.Recv(c.Context)
			defer release()

			// Get capability
			w := worker.Worker(f.Client())
			log.Info("Found new worker!")

			// Create tuple to insert into mapping
			workerTuple := &WorkerTuple{id: int(rand.Int63()), connections: 0, cap: w}
			workerMap.mu.Lock()
			workerMap.mapping[workerTuple.id] = workerTuple
			workerMap.mu.Unlock()

			time.Sleep(2 * time.Second)
		}
	}()

	http.HandleFunc("/tasks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			log.Info("Serving GET request")
			tasks.RLock()
			defer tasks.RUnlock()

			var taskList []Task
			for _, task := range tasks.m {
				taskList = append(taskList, task)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(taskList)

		case http.MethodPost:
			log.Info("Serving POST request")
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			var task Task
			err = json.Unmarshal(body, &task)
			if err != nil {
				http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
				return
			}

			tasks.Lock()
			tasks.m[task.ID] = task
			tasks.Unlock()

			go func() {
				// Function to execute the task
				executeTask := func() {
					// log.Infof("Starting task: %s", task.Description)
					// time.Sleep(time.Duration(task.Duration) * time.Second)
					// log.Infof("Task completed: %s", task.Description)

					// Get worker to execute task
					if len(workerMap.mapping) == 0 {
						// Handle the case when there's no worker available
						logger.Error("No worker available")
						return
					}

					// find smallest
					workerMap.mu.Lock()
					wTuple := findSmallestConnections()
					workerMap.mapping[wTuple.id].connections += 1
					workerMap.mu.Unlock()

					// Use capability to execute task
					logger.Info("Calling assign() method on worker #" + strconv.Itoa(wTuple.id) + " capability")
					worker, release := wTuple.cap.Assign(c.Context, worker.Data([]byte{}))
					defer release()

					// block until we get the RPC response
					res, err := worker.Struct()
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadGateway)
						return
					}
					logger.Info("We got response:" + res.String())

					// Update number of connections
					workerMap.mu.Lock()
					workerMap.mapping[wTuple.id].connections -= 1
					workerMap.mu.Unlock()

					// Update the Completions field after each successful execution
					tasks.Lock()
					task.Completions++
					tasks.m[task.ID] = task
					tasks.Unlock()
				}

				timer := time.NewTimer(time.Duration(task.Start) * time.Second)
				<-timer.C

				if task.Repeats > 1 && task.Delay > 0 {
					ticker := time.NewTicker(time.Duration(task.Delay) * time.Second)
					defer ticker.Stop()

					for i := 0; i < task.Repeats; i++ {
						executeTask()
						if i < task.Repeats-1 {
							<-ticker.C
						}
					}
				} else {
					executeTask()
				}
			}()

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(task)

		default:
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		}

	}))

	log.Info("starting server on port 8080")

	return http.ListenAndServe(":8080", nil)
}

// Assume holding the lock when going in
func findSmallestConnections() *WorkerTuple {
	var minConnections *WorkerTuple
	firstIteration := true

	for _, workerTuple := range workerMap.mapping {
		if firstIteration {
			minConnections = workerTuple
			firstIteration = false
		} else if workerTuple.connections < minConnections.connections {
			minConnections = workerTuple
		}
	}

	return minConnections
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

	// // Loop until the worker starts shutting down.
	// for c.Context.Done() == nil {
	// 	Setup of the worker capability. Start a worker server, and derive a client from it.
	server := worker.WorkerServer{}
	e := capnp.Client(worker.Worker_ServerToClient(server))

	// Block until we're able to send our worker capability to the
	// gateway server.  This is where the load-balancing happens.
	// We are competing with other Send()s, and the gateway will
	// pick one of the senders at random each time it handles an
	// HTTP request.
	logger.Info("Sending capability down gateway channel")
	err = ch.Send(context.Background(), csp.Client(e))
	if err != nil {
		return err // this generally means the gateway is down
	}

	<-c.Done() // TODO: Do we need to do this?

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
