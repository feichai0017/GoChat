package config

import "github.com/spf13/viper"

// GetDiscovName get discov using which method
func GetDiscovName() string {
	return viper.GetString("crpc.discov.name")
}

// GetDiscovEndpoints get discov endpoints
func GetDiscovEndpoints() []string {
	return viper.GetStringSlice("discovery.endpoints")
}

// GetTraceEnable whether to enable trace
func GetTraceEnable() bool {
	return viper.GetBool("crpc.trace.enable")
}

// GetTraceCollectionUrl get trace collection url
func GetTraceCollectionUrl() string {
	return viper.GetString("crpc.trace.url")
}

// GetTraceServiceName get service name
func GetTraceServiceName() string {
	return viper.GetString("crpc.trace.service_name")
}

// GetTraceSampler get trace sampler
func GetTraceSampler() float64 {
	return viper.GetFloat64("crpc.trace.sampler")
}