package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"time"
)

const (
	HeaderContentType = "Content-Type"
	ContentTypeJson   = "application/json"
)

type IClient interface {
	SetBaseURL(baseURL string)
	AddDefaultHeader(key, value string)
	Get(path string, queryParams url.Values, headers http.Header) (*Response, error)
	Post(path string, body any, queryParams url.Values, headers http.Header) (*Response, error)
	Patch(path string, body any, queryParams url.Values, headers http.Header) (*Response, error)
	Delete(path string, body any, queryParams url.Values, headers http.Header) (*Response, error)
	SendRequest(method, path string, body any, queryParams url.Values, headers http.Header) (*Response, error)
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type Client struct {
	client         *http.Client
	baseURL        string
	defaultHeaders http.Header
}

func NewHTTPClient(timeout time.Duration, allowRedirects bool, maxRedirects int) *Client {
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if allowRedirects {
				if len(via) > maxRedirects {
					return fmt.Errorf("too many redirects")
				}
				return nil
			}
			return http.ErrUseLastResponse
		},
	}
	return &Client{
		client:         client,
		defaultHeaders: make(http.Header),
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

func (c *Client) AddDefaultHeader(key, value string) {
	c.defaultHeaders.Add(key, value)
}

func (c *Client) Get(path string, queryParams url.Values, headers http.Header) (*Response, error) {
	return c.SendRequest("GET", path, nil, queryParams, headers)
}

func (c *Client) Post(path string, body any, queryParams url.Values, headers http.Header) (*Response, error) {
	return c.SendRequest("POST", path, body, queryParams, headers)
}

func (c *Client) Patch(path string, body any, queryParams url.Values, headers http.Header) (*Response, error) {
	return c.SendRequest("PATCH", path, body, queryParams, headers)
}

func (c *Client) Delete(path string, body any, queryParams url.Values, headers http.Header) (*Response, error) {
	return c.SendRequest("DELETE", path, body, queryParams, headers)
}

func (c *Client) SendRequest(method, path string, body any, queryParams url.Values, headers http.Header) (*Response, error) {
	var fullURL string
	if c.baseURL != "" {
		fullURL = c.baseURL + path
	} else {
		fullURL = path
	}

	if queryParams != nil {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		for k, vs := range queryParams {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	maps.Copy(req.Header, c.defaultHeaders)
	if headers != nil {
		maps.Copy(req.Header, headers)
	}

	if body != nil {
		var bodyReader io.Reader
		if strBody, ok := body.(string); ok {
			bodyReader = bytes.NewBufferString(strBody)
		} else if jsonBody, ok := body.(map[string]interface{}); ok {
			jsonData, err := json.Marshal(jsonBody)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonData)
			req.Header.Set(HeaderContentType, ContentTypeJson)
		} else if bytesBody, ok := body.([]byte); ok {
			bodyReader = bytes.NewReader(bytesBody)
		} else {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonData)
			req.Header.Set(HeaderContentType, ContentTypeJson)
		}
		req.Body = io.NopCloser(bodyReader)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %v", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header,
	}, nil
}
