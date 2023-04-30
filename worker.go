package worker

import (
	context "context"
	"time"
)

type WorkerServer struct{}

func (WorkerServer) Assign(ctx context.Context, call Worker_assign) error {
	res, err := call.AllocResults() // allocate the results struct
	if err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	// Set the response to just echo the input message
	_, err = call.Args().Wasm()
	if err != nil {
		return err
	}

	return res.SetResult("Ok")
}

func Data(b []byte) func(Worker_assign_Params) error {
	return func(call Worker_assign_Params) error {
		return call.SetWasm(b)
	}
}
