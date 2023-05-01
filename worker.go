package worker

import (
	context "context"
	"fmt"
	"log"
	"strconv"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type WorkerServer struct{}

func (WorkerServer) Assign(ctx context.Context, call Worker_assign) error {
	res, err := call.AllocResults() // allocate the results struct
	if err != nil {
		return err
	}

	input := call.Args().Input()
	if err != nil {
		return err
	}

	diff := call.Args().Difficulty()
	if err != nil {
		return err
	}

	fmt.Printf("Input: %d, Difficulty: %d => ", input, diff)

	hashWasm, err := call.Args().Wasm()
	if err != nil {
		return err
	}

	// Run the wasm process
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
	results, err := run.Call(ctx, uint64(input), uint64(diff))

	resStr := strconv.FormatUint(results[0], 10)

	fmt.Printf("Result: %s\n", resStr)

	return res.SetResult(resStr)
}

func Data(b []byte) func(Worker_assign_Params) error {
	return func(call Worker_assign_Params) error {
		return call.SetWasm(b)
	}
}
