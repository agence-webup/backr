package privatehttp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"webup/backr"
)

type PrivateAPIClient struct {
}

func NewClient() backr.PrivateAPIClient {
	return &PrivateAPIClient{}
}

func (client *PrivateAPIClient) Backup(projectName string) error {

	resp, err := http.Get("http://127.0.0.1:22258/actions/backup?name=" + projectName)
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
