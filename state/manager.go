package state

import (
	"fmt"
	"webup/backr"
	"webup/backr/bolt"
)

func GetStorage(opts backr.Settings) (backr.StateStorer, error) {
	if opts.StateStorage.GetType() == backr.StateStorageLocal {
		return bolt.GetStorage(opts)
	} else if opts.StateStorage.GetType() == backr.StateStorageEtcd {
		// return NewEtcdStorage(opts)
	}

	return nil, fmt.Errorf("Unable to detect the state storage")
}

func CleanupStorage(opts backr.Settings) {
	storer, err := GetStorage(opts)
	if err == nil {
		storer.Cleanup()
	}
}
