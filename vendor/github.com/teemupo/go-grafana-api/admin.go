package gapi

import (
	"errors"
	"fmt"
)

func (c *Client) DeleteUser(id int64) error {
	req, err := c.newRequest("DELETE", fmt.Sprintf("/api/admin/users/%d", id), nil)
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
	return err
}
