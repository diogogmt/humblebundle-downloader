package hbclient

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
)

const (
	baseURL       = "https://www.humblebundle.com/api/v1"
	jwtCookieName = "_simpleauth_sess"
)

type HBDClient struct {
	jwtCookie string
	apiURL    string
}

type HBClientOption = func(c *HBDClient)

// NewClient creates a client consuming the humble bundle HTTP API
func NewClient(opts ...HBClientOption) *HBDClient {
	client := HBDClient{
		apiURL: baseURL,
	}
	for _, opt := range opts {
		opt(&client)
	}
	return &client
}

// WithJWT sets a JWT cookie for the client
func WithJWT(jwtCookie string) HBClientOption {
	return func(c *HBDClient) {
		c.jwtCookie = jwtCookie
	}
}

// WithAPIURL overrides the default humble bundle API URL
func WithAPIURL(apiURL string) HBClientOption {
	return func(c *HBDClient) {
		c.apiURL = apiURL
	}
}

// GetOrder fetches an order details matching a given key
func (c *HBDClient) GetOrder(key string) (*Order, error) {
	u, err := url.Parse(c.apiURL)
	if err != nil {
		return nil, errors.Wrapf(err, "url.Parse baseURL %q", baseURL)
	}
	u.Path = path.Join(u.Path, "order")
	u.Path = path.Join(u.Path, key)
	// url; https://www.humblebundle.com/api/v1/order/Ms39KaHeZAZW6Xx7
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequestWithContext order")
	}
	if c.jwtCookie != "" {
		cookie := http.Cookie{
			Name:  jwtCookieName,
			Value: c.jwtCookie,
		}
		req.AddCookie(&cookie)
	}
	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "httpClient.Do get order")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ioutil.ReadAll order response")
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		hbError := HBError{}
		if err := json.Unmarshal(body, &hbError); err != nil {
			return nil, errors.Wrap(err, "json.Unmarshal error")
		}
		return nil, errors.Errorf("%s %s", hbError.Status, hbError.Message)
	}

	order := Order{}
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal order")
	}
	return &order, nil
}
