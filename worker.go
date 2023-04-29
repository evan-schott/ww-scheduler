package worker

import (
	context "context"
	"time"
)

type WorkerServer struct{}

func (WorkerServer) Worker(ctx context.Context, call Worker_assign) error {
	res, err := call.AllocResults() // allocate the results struct
	if err != nil {
		return err
	}

	time.Sleep(2 * time.Second)
	// Set the response to just echo the input message
	// msg, err := call.Args().Wasm()
	// if err != nil {
	// 	return err
	// }

	// // Decode object
	// var payload Payload
	// reader := bytes.NewReader(msg)
	// err = json.NewDecoder(reader).Decode(&payload)
	// if err != nil {
	// 	return err
	// }

	// payload.Message = payload.Message + " You have been echoed by worker!"
	// payload.Status = http.StatusOK
	// // payload.Headers = make(map[string]string)
	// // payload.Headers["Content-Type"] = "application/json"
	// responsePayload, err := json.Marshal(payload)

	return res.SetResult("Ok")
}

// func Data(b []byte) func(Echo_echo_Params) error {
// 	return func(call Echo_echo_Params) error {
// 		return call.SetPayload(b)
// 	}
// }
