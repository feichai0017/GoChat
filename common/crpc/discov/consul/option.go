package consul

import (
	"time"
)

var (
	defaultOption = Options{
		endpoints:               []string{"127.0.0.1:8500"},
		dialTimeout:             10 * time.Second,
		syncFlushCacheInterval:  10 * time.Second,
		registerServiceInterval: 10 * time.Second,
	}
)

type Options struct {
	syncFlushCacheInterval  time.Duration
	endpoints               []string
	dialTimeout             time.Duration
	registerServiceInterval time.Duration
}

type Option func(o *Options)

// WithEndpoints ...
func WithEndpoints(endpoints []string) Option {
	return func(o *Options) {
		o.endpoints = endpoints
	}
}

// WithDialTimeout ...
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *Options) {
		o.dialTimeout = dialTimeout
	}
}

// WithSyncFlushCacheInterval ...
func WithSyncFlushCacheInterval(t time.Duration) Option {
	return func(o *Options) {
		o.syncFlushCacheInterval = t
	}
}

// WithRegisterServiceInterval ...
func WithRegisterServiceInterval(t time.Duration) Option {
	return func(o *Options) {
		o.registerServiceInterval = t
	}
}
