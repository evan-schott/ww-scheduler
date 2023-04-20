package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/evan-schott/ww-load-balancer/handler"
	"github.com/stretchr/testify/require"
)

type Payload struct {
	Message string `json:"message"`
}

// need to get a valid http request.

func TestSync(t *testing.T) {
	t.Parallel()

	// Create dummy request
	p := []byte(`{"message": "Hello, world!"}`)
	dummy, err := http.NewRequest("POST", "/", bytes.NewBuffer(p))

	p1, p2 := net.Pipe()
	go func() {
		server := handler.HandlerServer{}
		client := handler.Handler_ServerToClient(server)

		conn := rpc.NewConn(rpc.NewStreamTransport(p1), &rpc.Options{
			BootstrapClient: capnp.Client(client),
		})
		defer conn.Close()
		<-conn.Done()
	}()

	conn := rpc.NewConn(rpc.NewStreamTransport(p2), nil)
	defer conn.Close()

	a := handler.Handler(conn.Bootstrap(context.Background()))

	// unmarshall dummy
	var payload Payload
	err = json.NewDecoder(dummy.Body).Decode(&payload)
	require.NoError(t, err)
	require.NotEqual(t, "", payload.Message)

	fmt.Println("Sending: " + payload.Message)
	f, release := a.Handle(context.Background(), func(ps handler.Handler_handle_Params) error {
		ps.SetRequest(payload.Message)
		return nil
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		panic(err)
	}

	resp, err := res.Response()
	fmt.Println(resp)

	// re-Marshall
	payload.Message = resp
	responsePayload, err := json.Marshal(payload)

	require.NoError(t, err)
	require.Equal(t, "Hello, world! You have been echoed by worker!", resp)

	fmt.Println(responsePayload)

}
