package config

import (
	"time"

	"github.com/spf13/viper"
)

func Init(path string) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

// get endpoints for discovery
func GetEndpointsForDiscovery() []string {
	return viper.GetStringSlice("discovery.endpoints")
}

// get timeout for discovery 
func GetTimeoutForDiscovery() time.Duration {
	return viper.GetDuration("discovery.timeout") * time.Second
}

func GetServicePathForIPConf() string {
	return viper.GetString("ip_conf.service_path")
}

// check if debug environment
func IsDebug() bool {
	env := viper.GetString("global.env")
	return env == "debug"
}