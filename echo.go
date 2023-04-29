package lb

import (
	"bytes"
	context "context"
	"encoding/json"
	"net/http"
)

// ArithServer satisfies the Arith_Server interface that was generated
// by the capnp compiler.
type EchoServer struct{}

type Payload struct {
	Message string            `json:"message"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// Multiply is the concrete implementation of the Multiply method that was
// defined in the schema. Notice that the method signature matches that of
// the Arith_Server interface.
//
// The Arith_multiply struct was generated by the capnp compiler.  You will
// find it in arith.capnp.go
func (EchoServer) Echo(ctx context.Context, call Echo_echo) error {
	res, err := call.AllocResults() // allocate the results struct
	if err != nil {
		return err
	}

	// Set the response to just echo the input message
	msg, err := call.Args().Payload()
	if err != nil {
		return err
	}

	// Decode object
	var payload Payload
	reader := bytes.NewReader(msg)
	err = json.NewDecoder(reader).Decode(&payload)
	if err != nil {
		return err
	}

	payload.Message = payload.Message + " You have been echoed by worker!"
	payload.Status = http.StatusOK
	// payload.Headers = make(map[string]string)
	// payload.Headers["Content-Type"] = "application/json"
	responsePayload, err := json.Marshal(payload)

	return res.SetResponse(responsePayload)
}

func Data(b []byte) func(Echo_echo_Params) error {
	return func(call Echo_echo_Params) error {
		return call.SetPayload(b)
	}
}
