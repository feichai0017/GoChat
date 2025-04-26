package source

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/feichai0017/GoChat/common/config"
	"github.com/feichai0017/GoChat/common/discovery"
)

// testServiceRegister creates a mock service register for testing
func testServiceRegister(ctx *context.Context, port string, node string) {
	// Create EndpointInfo with generic type 'any' to hold float64 values
	ed := &discovery.EndpointInfo[any]{
		IP:   "127.0.0.1",
		Port: port,
		// Ensure metadata values are stored as float64
		MetaData: map[string]any{
			"connect_num":   float64(rand.Intn(1000)),
			"message_bytes": float64(rand.Intn(100000)),
		},
	}

	key := fmt.Sprintf("%s/%s", config.GetServicePathForIPConf(), node)
	// Instantiate ServiceRegister with the correct type parameter [any]
	sre, err := discovery.NewServiceRegister(ctx, key, ed, 5)
	if err != nil {
		panic(err)
	}
	go sre.ListenLeaseRespChan()

	go func() {
		count := 0
		timer := time.NewTicker(time.Second * 2)
		for range timer.C {
			count++
			// Update with EndpointInfo[any] and float64 values
			ed := &discovery.EndpointInfo[any]{
				IP:   "127.0.0.1",
				Port: port,
				MetaData: map[string]any{
					"connect_num":   float64(rand.Intn(1000)),
					"message_bytes": float64(rand.Intn(100000)),
				},
			}
			sre.UpdateValue(ed)
		}
	}()
}
