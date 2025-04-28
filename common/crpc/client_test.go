package crpc

import (
	"testing"

	"github.com/feichai0017/GoChat/common/config"

	ptrace "github.com/feichai0017/GoChat/common/crpc/trace"
	"github.com/stretchr/testify/assert"
)

func TestNewPClient(t *testing.T) {
	config.Init("../../gochat.yaml")
	ptrace.StartAgent()
	defer ptrace.StopAgent()

	_, err := NewPClient("crpc_server")
	assert.NoError(t, err)
}
