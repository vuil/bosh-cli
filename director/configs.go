package director

import (
	"encoding/json"
	"fmt"
	"net/http"

	gourl "net/url"

	"strconv"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type Config struct {
	ID        string
	Name      string
	Type      string
	CreatedAt string
	Teams     []string
	Content   string
}

type ConfigsFilter struct {
	Type            string
	Name            string
	IncludeOutdated bool
}

type UpdateConfigBody struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (d DirectorImpl) LatestConfig(configType string, name string) (Config, error) {
	resps, err := d.client.latestConfig(configType, name)

	if err != nil {
		return Config{}, err
	}

	if len(resps) == 0 {
		return Config{}, bosherr.Error("No config")
	}

	return resps[0], nil
}

func (d DirectorImpl) LatestConfigByID(configID string) (Config, error) {
	return d.client.latestConfigByID(configID)
}

func (c Client) latestConfigByID(configID string) (Config, error) {
	var config Config

	query := gourl.Values{}
	query.Add("/", configID)
	path := fmt.Sprintf("/configs/%s", configID)

	respBody, response, err := c.clientRequest.RawGet(path, nil, nil)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return config, bosherr.WrapErrorf(err, "No config")
		}
		return config, bosherr.WrapErrorf(err, "Finding config")
	}

	err = json.Unmarshal(respBody, &config)

	if err != nil {
		return config, bosherr.WrapError(err, "Unmarshaling Director response")
	}

	return config, nil
}

func (d DirectorImpl) ListConfigs(filter ConfigsFilter) ([]Config, error) {
	return d.client.listConfigs(filter)
}

func (d DirectorImpl) UpdateConfig(configType string, name string, content []byte) error {
	body, err := json.Marshal(UpdateConfigBody{configType, name, string(content)})
	if err != nil {
		return bosherr.WrapError(err, "Can't marshal request body")
	}
	return d.client.updateConfig(body)
}

func (d DirectorImpl) DeleteConfig(configType string, name string) (bool, error) {
	return d.client.deleteConfig(configType, name)
}

func (c Client) latestConfig(configType string, name string) ([]Config, error) {
	var resps []Config

	query := gourl.Values{}
	query.Add("type", configType)
	query.Add("name", name)
	query.Add("latest", "true")
	path := fmt.Sprintf("/configs?%s", query.Encode())

	err := c.clientRequest.Get(path, &resps)
	if err != nil {
		return resps, bosherr.WrapError(err, "Finding config")
	}

	return resps, nil
}

func (c Client) listConfigs(filter ConfigsFilter) ([]Config, error) {
	var resps []Config

	query := gourl.Values{}
	query.Add("latest", strconv.FormatBool(!filter.IncludeOutdated))
	if filter.Type != "" {
		query.Add("type", filter.Type)
	}
	if filter.Name != "" {
		query.Add("name", filter.Name)
	}
	path := fmt.Sprintf("/configs?%s", query.Encode())

	err := c.clientRequest.Get(path, &resps)
	if err != nil {
		return resps, bosherr.WrapErrorf(err, "Listing configs")
	}

	return resps, nil
}

func (c Client) updateConfig(content []byte) error {
	setHeaders := func(req *http.Request) {
		req.Header.Add("Content-Type", "application/json")
	}

	_, _, err := c.clientRequest.RawPost("/configs", content, setHeaders)
	if err != nil {
		return bosherr.WrapErrorf(err, "Updating config")
	}

	return nil
}

func (c Client) deleteConfig(configType string, name string) (bool, error) {
	query := gourl.Values{}
	query.Add("type", configType)
	query.Add("name", name)
	path := fmt.Sprintf("/configs?%s", query.Encode())

	_, response, err := c.clientRequest.RawDelete(path)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, bosherr.WrapErrorf(err, "Deleting config")
	}

	return true, nil
}

func (d DirectorImpl) DiffConfig(configType string, name string, manifest []byte) (ConfigDiff, error) {
	body, err := json.Marshal(UpdateConfigBody{configType, name, string(manifest)})
	if err != nil {
		return ConfigDiff{}, bosherr.WrapError(err, "Can't marshal request body")
	}
	resp, err := d.client.DiffConfig(body)
	if err != nil {
		return ConfigDiff{}, err
	}

	return NewConfigDiff(resp.Diff), nil
}

func (c Client) DiffConfig(manifest []byte) (ConfigDiffResponse, error) {
	setHeaders := func(req *http.Request) {
		req.Header.Add("Content-Type", "application/json")
	}
	return c.postConfigDiff("/configs/diff", manifest, setHeaders)
}
