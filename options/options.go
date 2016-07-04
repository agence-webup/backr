package options

import (
	"time"

	"golang.org/x/net/context"
)

type key int

const optionsKey key = 0

// Options represents the settings that can be configured with CLI
type Options struct {
	EtcdEndpoints      []string
	WatchDirs          []string
	BackupRootDir      string
	TimeSpec           BackupTimeSpec
	SwiftUploadEnabled bool
	Swift              SwiftOptions
}

// SwiftOptions represents the settings needed to use Swift
type SwiftOptions struct {
	AuthURL       string
	User          string
	APIKey        string
	TenantName    string
	ContainerName string
}

// BackupTimeSpec specifies the time options for performing backup
type BackupTimeSpec struct {
	Hour   int
	Minute int
	Period time.Duration
}

// NewDefaultOptions returns default options
func NewDefaultOptions() Options {
	return Options{
		BackupRootDir:      "/backups",
		SwiftUploadEnabled: false,
		TimeSpec: BackupTimeSpec{
			Hour:   1,
			Minute: 0,
			Period: time.Duration(24) * time.Hour, // unit of 1 day for ttl and minAge (WARNING: cannot be less (scheduling issues))
		},
	}
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
