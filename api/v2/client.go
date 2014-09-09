package apiv2

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmcvetta/napping"
)

const V2URL = "https://api.digitalocean.com/v2/"

type Client struct {
	RootURL     string
	AccessToken string
}

type APIErrorResponse struct {
	Message string
	Errors  []struct {
		Resource string
		Field    string
		Code     string
	}
}

func NewClient(token string) *Client {
	return &Client{
		RootURL:     V2URL,
		AccessToken: token,
	}
}

func (c *Client) URL(name string) string {
	return fmt.Sprintf("%s%s", c.RootURL, name)
}

func (c *Client) Get(url string, p *napping.Params, result, errMsg interface{}) *APIErrorResponse {
	resp, err := c.Session().Get(c.URL(url), p, result, errMsg)
	if err != nil {
		return &APIErrorResponse{Message: err.Error()}
	}

	status := resp.Status() - (resp.Status() % 200)

	if status != http.StatusOK {
		return NewAPIErrorResponse(resp.RawText())
	}
	return nil
}

func (c *Client) Post(url string, payload, result, errMsg interface{}) *APIErrorResponse {
	resp, err := c.Session().Post(c.URL(url), payload, result, errMsg)
	if err != nil {
		return &APIErrorResponse{Message: err.Error()}
	}

	status := resp.Status() - (resp.Status() % 200)

	if status != http.StatusOK {
		return NewAPIErrorResponse(resp.RawText())
	}
	return nil
}

func (c *Client) Put(url string, payload, result, errMsg interface{}) *APIErrorResponse {
	resp, err := c.Session().Put(c.URL(url), payload, result, errMsg)
	if err != nil {
		return &APIErrorResponse{Message: err.Error()}
	}

	status := resp.Status() - (resp.Status() % 200)

	if status != 2 {
		return NewAPIErrorResponse(resp.RawText())
	}
	return nil
}

func (c *Client) Delete(url string, result, errMsg interface{}) *APIErrorResponse {
	resp, err := c.Session().Delete(c.URL(url), result, errMsg)
	if err != nil {
		return &APIErrorResponse{Message: err.Error()}
	}

	status := resp.Status() - (resp.Status() % 200)

	if status != http.StatusOK {
		return NewAPIErrorResponse(resp.RawText())
	}
	return nil
}

func (c *Client) Session() *napping.Session {
	header := &http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

	return &napping.Session{
		Header: header,
		Log:    false,
	}
}

func NewAPIErrorResponse(data string) *APIErrorResponse {
	aer := &APIErrorResponse{}
	if err := json.Unmarshal([]byte(data), aer); err != nil {
		panic(err)
	}
	return aer
}
