package tasks

import (
	"context"
	"fmt"
	"webup/backr"
	"webup/backr/swift"

	log "github.com/sirupsen/logrus"
)

func GetBackups(name string, ctx context.Context) ([]backr.UploadedArchiveInfo, error) {

	opts, ok := backr.SettingsFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("Unable to get options from context")
	}

	if opts.Swift == nil {
		return nil, fmt.Errorf("Swift is not enabled")
	}

	results, err := swift.Get(name, *opts.Swift)
	if err != nil {
		if richErr, ok := err.(backr.UploadedArchiveError); ok {
			log.Errorln(err)
			if richErr.IsFatal {
				return nil, richErr
			}
			return nil, nil
		}

		return nil, err
	}

	return results, nil
}
