package kfp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL         *url.URL
	httpClient      *http.Client
	pageSize        int
	bearerTokenFile string
}

type Run struct {
	ID        string
	Namespace string
	State     string
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithBearerTokenFile(file string) Option {
	return func(c *Client) {
		c.bearerTokenFile = file
	}
}

func NewClient(baseURL string, pageSize int, requestTimeout time.Duration, opts ...Option) (*Client, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("page size must be greater than zero")
	}

	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("base url must be absolute")
	}

	client := &Client{
		baseURL:    parsed,
		pageSize:   pageSize,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}

func (c *Client) ListRuns(ctx context.Context, namespace string) ([]Run, int, error) {
	var runs []Run
	requests := 0
	pageToken := ""

	for {
		reqURL := c.runsURL(namespace, pageToken)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, requests, fmt.Errorf("create request: %w", err)
		}
		if err := c.addAuth(req); err != nil {
			return nil, requests, err
		}

		resp, err := c.httpClient.Do(req)
		requests++
		if err != nil {
			return nil, requests, fmt.Errorf("list runs: %w", err)
		}

		var body listRunsResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&body)
		closeErr := resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, requests, fmt.Errorf("list runs: unexpected status %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, requests, fmt.Errorf("decode runs response: %w", decodeErr)
		}
		if closeErr != nil {
			return nil, requests, fmt.Errorf("close response body: %w", closeErr)
		}

		for _, run := range body.Runs {
			runs = append(runs, Run{
				ID:        firstNonEmpty(run.RunID, run.ID, run.UUID),
				Namespace: firstNonEmpty(run.Namespace, namespace, "unknown"),
				State:     normalizeState(run.State),
			})
		}

		pageToken = firstNonEmpty(body.NextPageToken, body.NextPageTokenSnake)
		if pageToken == "" {
			break
		}
	}

	return runs, requests, nil
}

func (c *Client) runsURL(namespace, pageToken string) string {
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, "/apis/v2beta1/runs")

	q := u.Query()
	q.Set("page_size", strconv.Itoa(c.pageSize))
	if namespace != "" {
		q.Set("namespace", namespace)
	}
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func (c *Client) addAuth(req *http.Request) error {
	if c.bearerTokenFile == "" {
		return nil
	}
	token, err := os.ReadFile(c.bearerTokenFile)
	if err != nil {
		return fmt.Errorf("read bearer token file: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	return nil
}

type listRunsResponse struct {
	Runs               []runResponse `json:"runs"`
	NextPageToken      string        `json:"nextPageToken"`
	NextPageTokenSnake string        `json:"next_page_token"`
}

type runResponse struct {
	RunID     string `json:"runId"`
	ID        string `json:"id"`
	UUID      string `json:"uuid"`
	Namespace string `json:"namespace"`
	State     string `json:"state"`
}

func normalizeState(state string) string {
	state = strings.TrimSpace(state)
	state = strings.TrimPrefix(state, "RUNTIME_STATE_")
	if state == "" || state == "UNSPECIFIED" {
		return "UNKNOWN"
	}
	return state
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
