
# Introduction

This repository demonstrates how to use Wetware to write distributed systems. It contains a simple application that schedules jobs for execution in a distributed worker pool. A single scheduler node exposes a REST API for scheduling “tasks”, and distributes them pseudorandomly to workers via Cap’n Proto RPC. The scheduler discovers worker nodes using Wetware’s zero-conf clustering API, and can adapt to works joining and leaving the network dynamically.

# Running it locally

## Installation
1. Clone the repository:
```
git clone https://github.com/evan-schott/ww-scheduler.git
```

## First Run
1. Build the single application binary:
```
cd ww-scheduler && go build -o ww-sched cmd/scheduler/main.go
```
2. Start the gateway node:
```
    ./ww-sched --gateway
    #Example Output: 12D3KooWKggVqaYBwJvCCFNV8zNqCQS6rbSMKTafjmMBH2pZpZ8t
```
3.  Start a worker and add it to the gateway’s pool
```
./ww-sched --dial 12D3KooWKggVqaYBwJvCCFNV8zNqCQS6rbSMKTafjmMBH2pZpZ8t
```

The application is now deployed and ready to service requests.

Under the hood, Wetware has automatically provided the following:

-   **Peer discovery:** Wetware peers automatically find each other via a configurable bootstrap system. By default, peers locate each other using a protocol that is similar to [mDNS](https://en.wikipedia.org/wiki/Multicast_DNS), but more scalable.
-   **Clustering:** Wetware peers automatically connect to form an efficient overlay network, which they use to maintain mutual awareness. Each peer is able to detect the presence of other peers, and negotiate network connections.
-   **Routing:** Peers can route data and network connections to each other, using each peer’s unique identifier.
-   **Services:** Wetware provides a set of cluster-oriented services upon which to build applications. The present demo makes use of Wetware’s native support for [Cap’n Proto’s RPC](https://capnproto.org/rpc.html), which provides a compact data encoding, a latency-aware RPC API, and security through object-capabilities.

# Architecture
## Bootstrapping
A node joins the wetware cluster by bootstrapping to a known peer in the network. This procedure involves sending and listening for discover packets in the multicast group specified in the header flags. In order to view all the packets you download [casm](https://github.com/wetware/casm) and run `casm cluster ls` .

<details><summary>Example Output</summary>

  

```

[0000] Got multicast packet (171 byte)
type: survey
namespace: ww
size: 171 bytes
peer: 12D3KooWKggVqaYBwJvCCFNV8zNqCQS6rbSMKTafjmMBH2pZpZ8t
distance: 255

[0000] Got multicast packet (217 byte)
type: response
namespace: ww
size: 217 bytes
peer: 12D3KooWQdc1sHWeBqbWuCPZEG5gjLy4tWrc8ppKYTRAMq4upeTP

[0006] Got multicast packet (171 byte)
type: survey
namespace: ww
size: 171 bytes
peer: 12D3KooWQdc1sHWeBqbWuCPZEG5gjLy4tWrc8ppKYTRAMq4upeTP
distance: 255
  
[0006] Got multicast packet (217 byte)
type: response
namespace: ww
size: 217 bytes
peer: 12D3KooWKggVqaYBwJvCCFNV8zNqCQS6rbSMKTafjmMBH2pZpZ8t

[0009] Got multicast packet (171 byte)
type: survey
namespace: ww
size: 171 bytes
peer: 12D3KooWKggVqaYBwJvCCFNV8zNqCQS6rbSMKTafjmMBH2pZpZ8t
distance: 255

[0009] Got multicast packet (217 byte)
type: response
namespace: ww
size: 217 bytes
peer: 12D3KooWQdc1sHWeBqbWuCPZEG5gjLy4tWrc8ppKYTRAMq4upeTP
```
</details>


## Capabilities
Capabilities are fundamental to the access control model of Wetware. A process must own a capability from another process in order to call RPCs. Capabilities can be broadcasted as demonstrated with:
```go
// Create synchronous channel server on Gateway server and export it
// Now any node on network can send to our channel server by using the channel capability
ch := csp.NewChan(&csp.SyncChan{})
defer ch.Close(c.Context)
n.Vat.Export(chanCap, chanProvider{ch})
```

Capabilities can also be sent privately by way of transmission through a channel.

```go
// Setup of the worker capability. Start a worker server, and derive a client from it.
server := worker.WorkerServer{}

// Block until we're able to send our worker capability to the gateway server.
e := capnp.Client(worker.Worker_ServerToClient(server))
err = ch.Send(context.Background(), csp.Client(e))

```
  
## Channel Server

The channel server is a fundamental primitive for interprocess communication (IPC). Here are the primary differences between using PubSub and using channels for IPC:

| Property | Channel | PubSub
| ------------- |:-------------:|:-----:|
| Reliability | Every object sent to the channel server will be held onto until it is received. | Packets will be dropped once buffer capacity is hit. |
| Ordering | If a receive request arrives at the channel server before another receive request, then it will return a value from a send request that arrived to the channel before the send request value returned to the later receive request | No ordering guarantees. |
| Synchrony | A call to send an object to the channel server will block until the value is successfully received. Likewise a call to receive an object from the channel will block until an object is available. | No synchrony guarantees. |
| Medium | Uses unicast | Uses multicast |
  
## Layering (Libp2p, Cap'n Proto, Casm, Wetware)
Libp2p: (Networking stack)
- "Host" abstraction to represent node in network.
- "Connections" abstraction of transport layer link between two peers that can support multiple "Streams" on top of them to represent different communication topics.

Cap'n Proto: (RPC protocol)
- "Capabilities" abstraction to represent permission for making calls on remote objects.
- Architecture for configuring dynamic configuration of capabilities between hosts.
- Efficient serialization scheme
- Promise pipelining

Casm: (Low level cluster primitives)
- Merger between Cap'n Proto and Libp2p layers to allow for capabilities sharing on cluster of Libp2p Hosts.
- Bootstrap protocol for joining cluster to peers on cluster and forming cluster.

Wetware: (Cluster middleware)
- Higher level primitives for IPC (PubSub and Channels), shared memory (Anchors), processes, and querying information about cluster state.

  
## WASM

In this example, the "tasks" that can be scheduled on worker nodes take the form of hash puzzles. But really any .wasm file can be used as a task. The directions below outline how the tasks are compiled, and then how to make requests to the cluster to execute the tasks.

```

// To compile hash.go into a .wasm representation
tinygo build -o wasm/hash.wasm -target=wasi hash.go
  
// To schedule a difficulty 3 hash puzzle with seed "0", that will start 1 second after it is received, and repeat every 2 seconds for 3 iterations.
curl -X POST -H "Content-Type: multipart/form-data" \
-F "id=task1" \
-F "description=Hash puzzle" \
-F "complete=0" \
-F "duration=5" \
-F "start=1" \
-F "delay=2" \
-F "repeats=3" \
-F "wasm=@hash.wasm" \
-F "input=0" \
-F "difficulty=3" \
http://localhost:8080/tasks

  
// To see the current state of jobs, and their progress towards completion
curl -X GET http://localhost:8080/tasks

```

  ##  Application Architecture
-   Gateway node:
    -   Services used
        -   Boot
        -   Clustering
        -   Routing
        -   RPC
        -   CSP
    -   Application layer
        -   HTTP REST server
        -   Task Scheduling logic
-   Worker node
    -   Services used
        -   Boot
        -   Clustering
        -   Routing
        -   RPC
        -   CSP
    -   Application layer
        -   Cap’n Proto `Worker` capability
            -   Executes WASM payload when told to by gateway node
        -   WASM runtime
            -   Executes arbitrary WASM bytecode

# In action
- Video showcase: https://www.youtube.com/watch?v=SwjcH1dP7fk


# Running it with docker
```
docker build -t ww-scheduler .
docker compose up

// In one shell to run gateway: (output will contain gateway peerID)
go run cmd/scheduler/main.go --gateway true

// In other shells to run workers
go run cmd/scheduler/main.go --dial [INSERT GATEWAY PEERID] 
```