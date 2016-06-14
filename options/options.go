package options

import "golang.org/x/net/context"

type key int

const optionsKey key = 0

// Options represents the settings that can be configured with CLI
type Options struct {
	EtcdEndpoints []string
	WatchDirs     []string
	BackupRootDir string
	StartHour     int
	Swift         SwiftOptions
}

// SwiftOptions represents the settings needed to use Swift
type SwiftOptions struct {
	AuthURL    string
	User       string
	APIKey     string
	TenantName string
}

// NewContext returns a context with associated options
func NewContext(ctx context.Context, options Options) context.Context {
	return context.WithValue(ctx, optionsKey, options)
}

// FromContext returns the options associated to a context
func FromContext(ctx context.Context) (Options, bool) {
	options, ok := ctx.Value(optionsKey).(Options)
	return options, ok
}
