package kfp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListRunsAllVisibleWithPagination(t *testing.T) {
	transport := &fakeTransport{
		responses: []string{
			`{"runs":[{"runId":"r1","namespace":"team-a","state":"RUNNING"}],"nextPageToken":"next"}`,
			`{"runs":[{"runId":"r2","namespace":"team-b","state":"RUNTIME_STATE_PENDING"}]}`,
		},
	}

	client, err := NewClient("http://kfp.example", 50, time.Second, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	runs, requests, err := client.ListRuns(context.Background(), "")
	if err != nil {
		t.Fatalf("ListRuns returned error: %v", err)
	}

	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if len(runs) != 2 {
		t.Fatalf("len(runs) = %d, want 2", len(runs))
	}
	if runs[1].State != "PENDING" {
		t.Fatalf("state = %q, want PENDING", runs[1].State)
	}
	if got := transport.requests[0].URL.String(); got != "http://kfp.example/apis/v2beta1/runs?page_size=50" {
		t.Fatalf("first request = %q", got)
	}
}

func TestListRunsRestrictedNamespace(t *testing.T) {
	transport := &fakeTransport{
		responses: []string{`{"runs":[{"runId":"r1","state":"RUNNING"}]}`},
	}

	client, err := NewClient("http://kfp.example", 100, time.Second, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	runs, _, err := client.ListRuns(context.Background(), "team-a")
	if err != nil {
		t.Fatalf("ListRuns returned error: %v", err)
	}
	if runs[0].Namespace != "team-a" {
		t.Fatalf("namespace = %q, want team-a", runs[0].Namespace)
	}
	if got := transport.requests[0].URL.Query().Get("namespace"); got != "team-a" {
		t.Fatalf("namespace query = %q, want team-a", got)
	}
}

func TestListRunsReturnsHTTPError(t *testing.T) {
	transport := &fakeTransport{
		statuses: []int{http.StatusInternalServerError},
		responses: []string{
			`nope`,
		},
	}

	client, err := NewClient("http://kfp.example", 100, time.Second, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	_, _, err = client.ListRuns(context.Background(), "")
	if err == nil {
		t.Fatal("ListRuns returned nil error")
	}
}

type fakeTransport struct {
	requests  []*http.Request
	responses []string
	statuses  []int
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, req)
	index := len(f.requests) - 1

	status := http.StatusOK
	if index < len(f.statuses) && f.statuses[index] != 0 {
		status = f.statuses[index]
	}

	body := `{}`
	if index < len(f.responses) {
		body = f.responses[index]
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}
