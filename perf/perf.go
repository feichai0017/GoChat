package perf

import (
	"net"

	"github.com/feichai0017/GoChat/common/sdk"
)


var (
	TcpConnNum int32
)

func RunMain() {
	for range int(TcpConnNum) {
		sdk.NewChat(net.ParseIP("127.0.0.1"), 8900, "eric", "test", "test")
	}
}