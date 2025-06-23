package discovery

import (
	"context"
	"testing"
	"time"
)

func TestServiceDiscovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ser := NewServiceDiscovery(&ctx)
	defer ser.Close()

	go ser.WatchService("/web/", func(key, value string) {}, func(key, value string) {})
	go ser.WatchService("/gRPC/", func(key, value string) {}, func(key, value string) {})

	<-ctx.Done()
}
