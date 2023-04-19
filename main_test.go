package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	t.Parallel()

	app := createApp() // TODO: Having trouble debugging. Hope would be can run 'go run main.go --role 0' in terminal, and then click 'debug test' here and will be able to set breakpoint and step through runWorker() execution
	err := app.Run([]string{"", "--num-peers", "1", "ns", "ww", "discover", "/ip4/228.8.8.8/udp/8822/multicast/lo0", "num-peers", "2"})
	require.NoError(t, err)
}
