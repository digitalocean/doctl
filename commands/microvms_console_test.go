package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConsoleControlError(t *testing.T) {
	require.NoError(t, consoleControlError("idle_timeout", "x"))
	require.NoError(t, consoleControlError("session_max_duration", "x"))
	err := consoleControlError("client_unresponsive", "x")
	require.Error(t, err)
	require.Contains(t, err.Error(), "stopped responding")
	err = consoleControlError("dial_failed", "boom")
	require.EqualError(t, err, "console error (dial_failed): boom")
}
