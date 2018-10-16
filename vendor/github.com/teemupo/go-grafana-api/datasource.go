package gapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

type DataSource struct {
	Id     int64  `json:"id,omitempty"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Access string `json:"access"`

	Database string `json:"database,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`

	OrgId     int64 `json:"orgId,omitempty"`
	IsDefault bool  `json:"isDefault"`

	BasicAuth         bool   `json:"basicAuth"`
	BasicAuthUser     string `json:"basicAuthUser,omitempty"`
	BasicAuthPassword string `json:"basicAuthPassword,omitempty"`

	JSONData       JSONData       `json:"jsonData,omitempty"`
	SecureJSONData SecureJSONData `json:"secureJsonData,omitempty"`
}

// JSONData is a representation of the datasource `jsonData` property
type JSONData struct {
	AssumeRoleArn           string `json:"assumeRoleArn,omitempty"`
	AuthType                string `json:"authType,omitempty"`
	CustomMetricsNamespaces string `json:"customMetricsNamespaces,omitempty"`
	DefaultRegion           string `json:"defaultRegion,omitempty"`
}

// SecureJSONData is a representation of the datasource `secureJsonData` property
type SecureJSONData struct {
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
}

func (c *Client) NewDataSource(s *DataSource) (int64, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return 0, err
	}
	req, err := c.newRequest("POST", "/api/datasources", bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != 200 {
		return 0, errors.New(resp.Status)
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	result := struct {
		Id int64 `json:"id"`
	}{}
	err = json.Unmarshal(data, &result)
	return result.Id, err
}

func (c *Client) UpdateDataSource(s *DataSource) error {
	path := fmt.Sprintf("/api/datasources/%d", s.Id)
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	req, err := c.newRequest("PUT", path, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	return nil
}

func (c *Client) DataSource(id int64) (*DataSource, error) {
	path := fmt.Sprintf("/api/datasources/%d", id)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &DataSource{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *Client) DeleteDataSource(id int64) error {
	path := fmt.Sprintf("/api/datasources/%d", id)
	req, err := c.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	return nil
}
