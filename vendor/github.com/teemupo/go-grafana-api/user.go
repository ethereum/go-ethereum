package gapi

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type User struct {
	Id      int64
	Email   string
	Name    string
	Login   string
	IsAdmin bool
}

func (c *Client) Users() ([]User, error) {
	users := make([]User, 0)
	req, err := c.newRequest("GET", "/api/users", nil)
	if err != nil {
		return users, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return users, err
	}
	if resp.StatusCode != 200 {
		return users, errors.New(resp.Status)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return users, err
	}
	err = json.Unmarshal(data, &users)
	if err != nil {
		return users, err
	}
	return users, err
}
