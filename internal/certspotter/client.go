package certspotter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/go-querystring/query"
)

var (
	// ErrUnexpectedStatus is returned for status codes other than 2XX
	ErrUnexpectedStatus = errors.New("unexpected status")
)

var (
	// BaseURL is the base url for certspotter API endpoint.
	BaseURL = "https://api.certspotter.com/v1"
)

// Client is a certspotter API client.
type Client struct {
	cfg    *Config
	client *http.Client
	url    string
}

// Config is used for configuring the client.
type Config struct {
	Token     string
	UserAgent string
}

// DoOptions are options used when doing a request.
type DoOptions struct {
	Method     string
	Path       string
	Parameters interface{}
}

// NewClient returns a new certspotter API client.
func NewClient(cfg *Config) *Client {
	return &Client{
		cfg:    cfg,
		client: &http.Client{},
		url:    BaseURL,
	}
}

// GetURL returns a url string for path and parameters or errors.
func (c *Client) GetURL(path string, params interface{}) (string, error) {
	endpoint := fmt.Sprintf("%s/%s", c.url, path)
	url, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	vals, err := query.Values(params)
	if err != nil {
		return "", err
	}

	url.RawQuery = vals.Encode()
	return url.String(), nil
}

// Do sends a request with options to certspotter api and encodes json
// response into val.
func (c *Client) Do(ctx context.Context, val interface{}, opts *DoOptions) (*http.Response, error) {
	url, err := c.GetURL(opts.Path, opts.Parameters)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(opts.Method, url, nil)
	if err != nil {
		return nil, err
	}

	if c.cfg.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.Token))
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.cfg.UserAgent)

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if err := CheckResponse(resp); err != nil {
		return resp, err
	}

	return resp, json.NewDecoder(resp.Body).Decode(&val)
}

// CheckResponse returns an error if http.Response was unsuccessful.
func CheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode <= 399 {
		return nil
	}
	return fmt.Errorf("%w %s", ErrUnexpectedStatus, resp.Status)
}
