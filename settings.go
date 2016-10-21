package backr

import (
	"context"
	"time"
)

type key int

const settingsKey key = 0

// Settings represents the settings that can be configured with CLI
type Settings struct {
	StateStorage  StateStorageSettings
	WatchDirs     []string
	BackupRootDir string
	TimeSpec      BackupTimeSpec
	StartupTime   time.Time
	// ConfigRefreshRate  int
	// SwiftUploadEnabled bool
	Swift *SwiftSettings
}

// SwiftSettings represents the settings needed to use Swift
type SwiftSettings struct {
	AuthURL       string
	User          string
	APIKey        string
	TenantName    string
	ContainerName string
}

type StateStorageType string

const (
	StateStorageEtcd  StateStorageType = "etcd"
	StateStorageLocal StateStorageType = "local"
)

type StateStorageSettings struct {
	EtcdEndpoints *string
	LocalPath     *string
}

func (st *StateStorageSettings) GetType() StateStorageType {
	if st.LocalPath != nil && *st.LocalPath != "" {
		return StateStorageLocal
	}

	return StateStorageEtcd
}

// BackupTimeSpec specifies the time options for performing backup
type BackupTimeSpec struct {
	Hour   int
	Minute int
	Period time.Duration
}

// NewDefaultSettings returns default options
func NewDefaultSettings() Settings {
	return Settings{
		BackupRootDir: "backups",
		// SwiftUploadEnabled: false,
		TimeSpec: BackupTimeSpec{
			Hour:   1,
			Minute: 0,
			Period: time.Duration(24) * time.Hour, // unit of 1 day for ttl and minAge (WARNING: cannot be less (scheduling issues))
		},
		StartupTime: time.Now(),
	}
}

// NewContextWithSettings returns a context with associated options
func NewContextWithSettings(ctx context.Context, settings Settings) context.Context {
	return context.WithValue(ctx, settingsKey, settings)
}

// SettingsFromContext returns the options associated to a context
func SettingsFromContext(ctx context.Context) (Settings, bool) {
	options, ok := ctx.Value(settingsKey).(Settings)
	return options, ok
}
