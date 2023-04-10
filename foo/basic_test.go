package main

import (
	"context"
	"fmt"
	"net"
	"testing"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/evan-schott/ww-load-balancer/foo/echo"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	t.Parallel()

	p1, p2 := net.Pipe()
	go func() {
		server := echo.EchoServer{}
		client := echo.Echo_ServerToClient(server)

		conn := rpc.NewConn(rpc.NewStreamTransport(p1), &rpc.Options{
			BootstrapClient: capnp.Client(client),
		})
		defer conn.Close()
		<-conn.Done()
	}()

	conn := rpc.NewConn(rpc.NewStreamTransport(p2), nil)
	defer conn.Close()

	a := echo.Echo(conn.Bootstrap(context.Background()))

	f, release := a.Send(context.Background(), func(ps echo.Echo_send_Params) error {
		ps.SetMsg("hello!")
		return nil
	})
	defer release()

	res, err := f.Struct()
	if err != nil {
		panic(err)
	}

	resp, err := res.Response()
	fmt.Println(resp)

	require.Equal(t, "hello!", resp)
	require.NoError(t, err)

}
