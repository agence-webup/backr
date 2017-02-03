package privatehttp

import (
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

func (client *PrivateAPIClient) Backup(projectName string) error {

	resp, err := http.Get(client.URL + "/actions/backup?name=" + projectName)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%v", string(body))
	}

	return nil
}
