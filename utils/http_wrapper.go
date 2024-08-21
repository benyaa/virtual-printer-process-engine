package utils

import "net/http"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewHTTPClient() HTTPClient {
	return &DefaultClient{
		Client: &http.Client{},
	}
}

type DefaultClient struct {
	Client *http.Client
}

func (c *DefaultClient) Do(req *http.Request) (*http.Response, error) {
	return c.Client.Do(req)
}
