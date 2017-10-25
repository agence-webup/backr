package privatehttp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"webup/backr"
)

type PrivateAPIClient struct {
	URL string
}

func NewClient(URL string) backr.PrivateAPIClient {
	return &PrivateAPIClient{URL: URL}
}

func (client *PrivateAPIClient) Backup(projectName string) (*backr.UploadedArchiveInfo, error) {

	resp, err := http.Get(client.URL + "/actions/backup?name=" + projectName)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%v", string(body))
	}

	var info backr.UploadedArchiveInfo
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}
