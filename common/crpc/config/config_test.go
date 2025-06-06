package config

import (
	"fmt"
	"testing"

	"github.com/feichai0017/GoChat/common/config"
)

func TestMain(m *testing.M) {
	config.Init("../../../gochat.yaml")
	m.Run()
}

func TestGetDiscovName(t *testing.T) {
	fmt.Println(GetDiscovName())
}

func TestGetDiscovEndpoints(t *testing.T) {
	fmt.Println(GetDiscovEndpoints())
}

func TestGetTraceEnable(t *testing.T) {
	fmt.Println(GetTraceEnable())
}

func TestGetTraceCollectionUrl(t *testing.T) {
	fmt.Println(GetTraceEnable())
}

func TestGetTraceServiceName(t *testing.T) {
	fmt.Println(GetTraceServiceName())
}

func TestGetTraceSampler(t *testing.T) {
	fmt.Println(GetTraceSampler())
}