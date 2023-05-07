package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
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

// Flags used for cli.App()
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

// Formatting of the REST api "/tasks"
type Task struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completions int    `json:"complete"`
	Duration    int    `json:"duration"`
	Start       int    `json:"start"`
	Delay       int    `json:"delay"`
	Repeats     int    `json:"repeats"`
	Wasm        []byte `json:"wasm"`
	Input       int    `json:"input"`
	Difficulty  int    `json:"difficulty"`
}

// Each task is protected by mutex to prevent concurrent access
var tasks = struct {
	sync.RWMutex
	m map[string]Task
}{m: make(map[string]Task)}

// Representation of a worker in cluster
type WorkerTuple struct {
	id          int
	connections int
	cap         worker.Worker
	release     capnp.ReleaseFunc
}

// Worker information is protected by mutex to prevent concurrent access
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

	// Command line argument to specify which is "gateway" and which is "worker"
	if c.Bool("gateway") {
		return runGateway(c, n)
	}

	return runWorker(c, n)
}

// Abbreviated struct for responses
type TaskResponse struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completions int    `json:"completions"`
	Duration    int    `json:"duration"`
	Start       int    `json:"start"`
	Delay       int    `json:"delay"`
	Repeats     int    `json:"repeats"`
	Input       int    `json:"input"`
	Difficulty  int    `json:"difficulty"`
}

// Core function for "gateway" node
func runGateway(c *cli.Context, n *server.Node) error {
	log := logger.With(n)
	log.Info("Gateway started")

	// Create synchronous channel server on Gateway server and export it
	// Now any node on network can send to our channel server by using the channel capability
	ch := csp.NewChan(&csp.SyncChan{})
	defer ch.Close(c.Context)
	n.Vat.Export(chanCap, chanProvider{ch})

	fmt.Println("Exported channel")

	// Launch go routine to receive new workers from channel
	go func() {
		fmt.Println("Starting go routine to listen for new workers trying to join")

		// Loop until the gateway starts shutting down
		// This way we allow for dynamic worker set
		for c.Context.Done() == nil {

			// Block on channel until a new worker capability arrives
			f, release := ch.Recv(c.Context)
			<-f.Done()

			// Get capability
			w := worker.Worker(f.Client())

			// Create new tuple to represent worker to insert into mapping
			workerTuple := &WorkerTuple{id: int(rand.Int63()), connections: 0, cap: w, release: release}
			workerMap.mu.Lock()
			workerMap.mapping[workerTuple.id] = workerTuple
			workerMap.mu.Unlock()

			fmt.Println("Found worker " + strconv.Itoa(workerTuple.id))
		}
	}()

	// Function to handle requests to "/tasks" endpoint
	http.HandleFunc("/tasks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fmt.Println("Serving GET request")
			tasks.RLock()
			defer tasks.RUnlock()

			var taskList []TaskResponse
			for _, task := range tasks.m {
				taskList = append(taskList, TaskResponse{
					ID:          task.ID,
					Description: task.Description,
					Completions: task.Completions,
					Duration:    task.Duration,
					Start:       task.Start,
					Delay:       task.Delay,
					Repeats:     task.Repeats,
					Input:       task.Input,
					Difficulty:  task.Difficulty,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(taskList)

		case http.MethodPost:
			fmt.Println("Serving POST request")
			err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
			if err != nil {
				http.Error(w, "Error parsing multipart form data", http.StatusBadRequest)
				return
			}

			task := Task{
				ID:          r.FormValue("id"),
				Description: r.FormValue("description"),
				Completions: atoiOrZero(r.FormValue("complete")),
				Duration:    atoiOrZero(r.FormValue("duration")),
				Start:       atoiOrZero(r.FormValue("start")),
				Delay:       atoiOrZero(r.FormValue("delay")),
				Repeats:     atoiOrZero(r.FormValue("repeats")),
				Input:       atoiOrZero(r.FormValue("input")),
				Difficulty:  atoiOrZero(r.FormValue("difficulty")),
			}

			// Read wasm file
			wasmFile, _, err := r.FormFile("wasm")
			if err != nil {
				http.Error(w, "Error reading wasm file", http.StatusBadRequest)
				return
			}
			defer wasmFile.Close()

			task.Wasm, err = ioutil.ReadAll(wasmFile)
			if err != nil {
				http.Error(w, "Error reading wasm file", http.StatusInternalServerError)
				return
			}

			// Use locks to prevent concurrent request handling issues
			tasks.Lock()
			tasks.m[task.ID] = task
			tasks.Unlock()

			// Launch go routine to manage current task
			go func() {
				// Function to execute the task
				executeTask := func() {

					// Get worker to execute task
					if len(workerMap.mapping) == 0 {
						// Handle the case when there's no worker available
						logger.Error("No worker available")
						return
					}

					// Least connections heuristic for load balancing
					workerMap.mu.Lock()
					wTuple := findSmallestConnections()
					workerMap.mapping[wTuple.id].connections += 1
					workerMap.mu.Unlock()

					// Use worker capability (must have received from channel earlier) to execute task
					// This will make the worker run the wasm file, and send us the results
					fmt.Printf("Calling assign(input:%d,difficulty:%d,wasm:hash.wasm) method on worker #%d\n", task.Input, task.Difficulty, wTuple.id)
					worker, release := wTuple.cap.Assign(c.Context, func(ps worker.Worker_assign_Params) error {
						ps.SetInput(int64(task.Input))
						ps.SetDifficulty(int64(task.Difficulty))
						ps.SetWasm(task.Wasm)
						return nil
					})
					defer release()

					// Block until we get the RPC response
					res, err := worker.Struct()
					if err != nil {

						// Remove workers that crash
						fmt.Println("Worker crashed")
						workerMap.mu.Lock()
						delete(workerMap.mapping, wTuple.id)
						workerMap.mu.Unlock()
						http.Error(w, err.Error(), http.StatusBadGateway)
						return
					}

					msg, err := res.Result()
					fmt.Printf("Got result:%s from worker #%d\n", msg, wTuple.id)

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

				// Delay as specified by request
				timer := time.NewTimer(time.Duration(task.Start) * time.Second)
				<-timer.C

				// Repeat task at regular interval specified by request
				if task.Repeats > 1 && task.Delay > 0 {
					ticker := time.NewTicker(time.Duration(task.Delay) * time.Second)
					defer ticker.Stop()

					for i := 0; i < task.Repeats; i++ {
						go executeTask()
						if i < task.Repeats-1 {
							<-ticker.C
						}
					}
				} else {
					executeTask()
				}
			}()

			// Return abbreviated struct
			taskSend := Task{
				ID:          r.FormValue("id"),
				Description: r.FormValue("description"),
				Completions: atoiOrZero(r.FormValue("complete")),
				Duration:    atoiOrZero(r.FormValue("duration")),
				Start:       atoiOrZero(r.FormValue("start")),
				Delay:       atoiOrZero(r.FormValue("delay")),
				Repeats:     atoiOrZero(r.FormValue("repeats")),
				Input:       atoiOrZero(r.FormValue("input")),
				Difficulty:  atoiOrZero(r.FormValue("difficulty")),
			}

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(taskSend)

		default:
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		}

	}))

	// Start server to begin handling requests
	fmt.Println("starting server on port 8080")
	return http.ListenAndServe(":8080", nil)
}

// Assume holding the lock when going in
func findSmallestConnections() *WorkerTuple {

	var minConnections *WorkerTuple
	firstIteration := true
	var infoBuilder strings.Builder

	for key, workerTuple := range workerMap.mapping {
		// Append the key and connections field to the infoBuilder
		infoBuilder.WriteString(fmt.Sprintf("{Id: %d, Connections: %d} ", key, workerTuple.connections))

		if firstIteration {
			minConnections = workerTuple
			firstIteration = false
		} else if workerTuple.connections < minConnections.connections {
			minConnections = workerTuple
		}
	}

	// Print the information about keys and connections
	fmt.Printf("Picked worker %d from %s\n", minConnections.id, infoBuilder.String())

	return minConnections
}

// Helper function for runGateway
func atoiOrZero(s string) int {
	value, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return value
}

// Main "worker" function
func runWorker(c *cli.Context, n *server.Node) error {

	// Wait until find gateway as peer in cluster
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
	log.Info("Worker started")

	// Establish connection with gateway (corresponding to channel capability that gateway exported earlier)
	conn, err := n.Vat.Connect(c.Context, addr, chanCap)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Recover channel capability from Gateway.  Cast it as a send-only channel, to avoid mistakes.
	ch := csp.Sender(conn.Bootstrap(c.Context))

	// 	Setup of the worker capability. Start a worker server, and derive a client from it.
	server := worker.WorkerServer{}
	e := capnp.Client(worker.Worker_ServerToClient(server))

	fmt.Println("Sending capability down gateway channel")
	err = ch.Send(context.Background(), csp.Client(e))
	if err != nil {
		return err // this generally means the gateway is down
	}

	<-c.Done()

	return nil
}

// Repeatedly query to see if gateway is peer in cluster
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

// Iterate over all peers in cluster and see if one of them is gateway
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
