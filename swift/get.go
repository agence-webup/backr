package swift

import (
	"errors"
	"strconv"
	"time"
	"webup/backr"

	"github.com/ncw/swift"
)

func Get(query string, settings backr.SwiftSettings) ([]backr.UploadedArchiveInfo, error) {
	items := []backr.UploadedArchiveInfo{}

	// get a swift connection
	c, err := getSwiftConnection(settings)
	if err != nil {
		return items, backr.UploadedArchiveError{IsFatal: true, Err: err}
	}

	// fetch all backups with the name starting with the term passed as param
	objects, err := c.ObjectsAll(settings.ContainerName, &swift.ObjectsOpts{
		Prefix: query,
	})
	if err != nil {
		return items, backr.UploadedArchiveError{IsFatal: true, Err: err}
	}

	if len(objects) == 0 {
		return items, backr.UploadedArchiveError{IsFatal: false, Err: errors.New("No project or backup found.")}
	}

	// fetch the headers for the Account (allowing to get the key for temp urls)
	_, headers, accountErr := c.Account()

	results := make(chan backr.UploadedArchiveInfo)

	for _, obj := range objects {

		go func(obj swift.Object) {
			// prepare info for this backup
			info := backr.UploadedArchiveInfo{
				Name: obj.Name,
			}

			// get the expire time
			_, objHeaders, objErr := c.Object(settings.ContainerName, obj.Name)
			if objErr == nil {
				if deleteAt, ok := objHeaders["X-Delete-At"]; ok {
					timestamp, _ := strconv.ParseInt(deleteAt, 10, 64)
					expire := time.Unix(timestamp, 0)
					info.Expire = expire
				}
			}

			// generate a temp url for download
			if accountErr == nil {
				info.URL = c.ObjectTempUrl(settings.ContainerName, obj.Name, headers["X-Account-Meta-Temp-Url-Key"], "GET", time.Now().Add(2*time.Minute))
			}

			results <- info

		}(obj)
	}

	for i := 0; i < len(objects); i++ {
		info := <-results
		items = append(items, info)
	}

	return items, nil
}
