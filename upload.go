package backr

import (
	"fmt"
	"time"
)

type UploadedArchiveError struct {
	Err     error
	IsFatal bool
}

func (e UploadedArchiveError) Error() string {
	return e.Err.Error()
}

type UploadedArchiveInfo struct {
	Name   string
	Expire time.Time
	URL    string
}

func (info UploadedArchiveInfo) String() string {
	return fmt.Sprintf("    name: %s\n", info.Name) +
		fmt.Sprintf(" expires: %v\n", info.Expire) +
		fmt.Sprintf("     url: %s\n", info.URL)
}
