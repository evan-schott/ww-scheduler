package worker_test

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net"
	"testing"

	worker "github.com/evan-schott/ww-scheduler"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/stretchr/testify/require"
)

//go:embed hash.wasm
var hashWasm []byte

func TestSync(t *testing.T) {
	t.Parallel()

	input := int64(0)
	diff := int64(3)

	p1, p2 := net.Pipe()
	go func() {
		server := worker.WorkerServer{}
		client := worker.Worker_ServerToClient(server)

		conn := rpc.NewConn(rpc.NewStreamTransport(p1), &rpc.Options{
			BootstrapClient: capnp.Client(client),
		})
		defer conn.Close()
		<-conn.Done()
	}()

	conn := rpc.NewConn(rpc.NewStreamTransport(p2), nil)
	defer conn.Close()

	a := worker.Worker(conn.Bootstrap(context.Background()))

	f, release := a.Assign(context.Background(), func(ps worker.Worker_assign_Params) error {
		ps.SetInput(input)
		ps.SetDifficulty(diff)
		ps.SetWasm(hashWasm)
		return nil
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		panic(err)
	}

	msg, err := res.Result()
	fmt.Println("Return value:" + msg)

	require.NoError(t, err)
	require.Equal(t, "2832", msg)
}

func TestHashWasm(t *testing.T) {
	t.Parallel()

	// Choose the context to use for function calls.
	ctx := context.Background()

	// Create a new WebAssembly Runtime.
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx) // This closes everything this Runtime created.

	// Instantiate WASI, which implements host functions needed for TinyGo to
	// implement `panic`.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Instantiate the guest Wasm into the same runtime. It exports the `run`
	// function, implemented in WebAssembly.
	mod, err := r.Instantiate(ctx, hashWasm)
	if err != nil {
		log.Panicf("failed to instantiate module: %v", err)
	}

	// Call the `run` function and print the results to the console.
	run := mod.ExportedFunction("run")
	x := uint64(0)
	y := uint64(3)
	results, err := run.Call(ctx, x, y)
	fmt.Println(results)
	if err != nil {
		log.Panicf("failed to call run: %v", err)
	}

	fmt.Printf("%d", results[0])

	require.Equal(t, uint64(2832), results[0])
}
