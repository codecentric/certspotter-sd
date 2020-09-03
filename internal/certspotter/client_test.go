package certspotter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func setup() (*Client, *http.ServeMux, func()) {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	return &Client{
		cfg:    &Config{},
		client: &http.Client{},
		url:    ts.URL,
	}, mux, ts.Close
}

func TestClientGetURL(t *testing.T) {
	table := map[string]struct {
		path   string
		params interface{}
		want   string
	}{"only path": {
		"issuances",
		struct{}{},
		"issuances",
	}, "with params": {
		"issuances",
		struct {
			Domain            string `url:"domain"`
			IncludeSubdomains bool   `url:"include_subdomains"`
		}{
			"example.com", false,
		},
		"issuances?domain=example.com&include_subdomains=false",
	}}

	cl, _, stop := setup()
	defer stop()

	for name, test := range table {
		t.Logf("testing: %s", name)

		want := fmt.Sprintf("%s/%s", cl.url, test.want)
		got, err := cl.GetURL(test.path, test.params)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %q want %q", got, want)
		}
	}
}

type sample struct {
	ID string `json:"id"`
}

func TestClientDo(t *testing.T) {
	table := map[string]struct {
		data string
		want sample
	}{"empty sample": {
		`{}`,
		sample{},
	}, "complete sample": {
		`{"id": "1"}`,
		sample{ID: "1"},
	}}

	ctx := context.Background()
	cl, mux, stop := setup()
	defer stop()

	var tname string
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, table[tname].data)
	})

	opts := &DoOptions{Path: "/test"}

	for name, test := range table {
		t.Logf("testing: %s", name)

		tname = name
		var got sample
		if _, err := cl.Do(ctx, &got, opts); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %q want %q", got, test.want)
		}
	}
}

func TestCheckResponse(t *testing.T) {
	table := map[string]struct {
		resp *http.Response
		want error
	}{
		"status 200": {&http.Response{StatusCode: 200}, nil},
		"status 199": {&http.Response{StatusCode: 199}, ErrUnexpectedStatus},
		"status 400": {&http.Response{StatusCode: 400}, ErrUnexpectedStatus},
	}

	for name, test := range table {
		t.Logf("testing: %s", name)

		got := CheckResponse(test.resp)
		if !errors.Is(got, test.want) {
			t.Errorf("got: %q; want: %q", got, test.want)
		}
	}
}
