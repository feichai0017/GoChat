package k8s

import (
	"time"
)

var (
	defaultOption = Options{
		syncInterval: 10 * time.Second,
	}
)

type Options struct {
	syncInterval time.Duration
}

type Option func(o *Options)

// WithSyncInterval ...
func WithSyncInterval(t time.Duration) Option {
	return func(o *Options) {
		o.syncInterval = t
	}
}
