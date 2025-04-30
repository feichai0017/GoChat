package crpc

import (
	"testing"

	"github.com/feichai0017/GoChat/common/config"

	ctrace "github.com/feichai0017/GoChat/common/crpc/trace"
	"github.com/stretchr/testify/assert"
)

func TestNewCClient(t *testing.T) {
	config.Init("../../gochat.yaml")
	ctrace.StartAgent()
	defer ctrace.StopAgent()

	_, err := NewCClient("crpc_server")
	assert.NoError(t, err)
}
