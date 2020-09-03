package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
)

func setup() (*Client, *http.ServeMux, func()) {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	certspotter.BaseURL = ts.URL

	logger := zap.NewNop()
	client := NewClient(logger, &Config{
		Interval: 0,
	})
	return client, mux, ts.Close
}

func TestClientGetIssuances(t *testing.T) {
	table := map[string]struct {
		data map[string]string
		opts *certspotter.GetIssuancesOptions
		want []*certspotter.Issuance
	}{"zero pages": {
		map[string]string{
			"": `[]`,
		},
		&certspotter.GetIssuancesOptions{Domain: "example.com"},
		[]*certspotter.Issuance(nil),
	}, "single page": {
		map[string]string{
			"":          `[{"id":"648494876"}]`,
			"648494876": `[]`,
		},
		&certspotter.GetIssuancesOptions{Domain: "example.com"},
		[]*certspotter.Issuance{
			&certspotter.Issuance{ID: "648494876"},
		},
	}, "multiple pages": {
		map[string]string{
			"":          `[{"id":"648494876"}]`,
			"648494876": `[{"id":"648494877"}]`,
			"648494877": `[]`,
		},
		&certspotter.GetIssuancesOptions{Domain: "example.com"},
		[]*certspotter.Issuance{
			&certspotter.Issuance{ID: "648494876"},
			&certspotter.Issuance{ID: "648494877"},
		},
	}}

	ctx := context.Background()
	cl, mux, stop := setup()
	defer stop()

	var tname string
	mux.HandleFunc("/issuances", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		after := query.Get("after")
		data := table[tname].data[after]
		fmt.Fprint(w, data)
	})

	for name, test := range table {
		t.Logf("testing: %s", name)

		tname = name
		got, _, err := cl.GetIssuances(ctx, test.opts)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %#v want %#v", got, test.want)
		}
	}
}

func TestClientSubIssuances(t *testing.T) {
	table := map[string]struct {
		datas []map[string]string
		opts  *certspotter.GetIssuancesOptions
		want  [][]*certspotter.Issuance
	}{"zero new issuances": {
		[]map[string]string{
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[]`,
			},
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[]`,
			},
		},
		&certspotter.GetIssuancesOptions{Domain: "example.com"},
		[][]*certspotter.Issuance{
			[]*certspotter.Issuance{&certspotter.Issuance{ID: "648494876"}},
			[]*certspotter.Issuance(nil),
		},
	}, "single new issuances": {
		[]map[string]string{
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[]`,
			},
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[{"id":"648494877"}]`,
				"648494877": `[]`,
			},
		},
		&certspotter.GetIssuancesOptions{Domain: "example.com"},
		[][]*certspotter.Issuance{
			[]*certspotter.Issuance{&certspotter.Issuance{ID: "648494876"}},
			[]*certspotter.Issuance{&certspotter.Issuance{ID: "648494877"}},
		},
	}, "delayed new issuances": {
		[]map[string]string{
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[]`,
			},
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[]`,
			},
			map[string]string{
				"":          `[{"id":"648494876"}]`,
				"648494876": `[{"id":"648494877"}]`,
				"648494877": `[]`,
			},
		},
		&certspotter.GetIssuancesOptions{Domain: "example.com"},
		[][]*certspotter.Issuance{
			[]*certspotter.Issuance{&certspotter.Issuance{ID: "648494876"}},
			[]*certspotter.Issuance(nil),
			[]*certspotter.Issuance{&certspotter.Issuance{ID: "648494877"}},
		},
	}}

	read := func(tname string, num int) [][]*certspotter.Issuance {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cl, mux, stop := setup()
		defer stop()

		var idx int
		mux.HandleFunc("/issuances", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			after := query.Get("after")
			data := table[tname].datas[idx][after]
			w.Header().Add("Retry-After", "0")
			fmt.Fprint(w, data)
		})

		ch := cl.SubIssuances(ctx, table[tname].opts)
		var issuances [][]*certspotter.Issuance
		for ; idx < num; idx++ {
			issuances = append(issuances, <-ch)
		}
		return issuances
	}

	for name, test := range table {
		t.Logf("testing: %s", name)

		got := read(name, len(test.want))
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %v want: %v", got, test.want)
		}
	}
}

func TestGetRetryAfter(t *testing.T) {
	table := map[string]struct {
		resp *http.Response
		want time.Duration
		ok   bool
	}{"good header": {
		&http.Response{Header: map[string][]string{
			"Retry-After": []string{"3600"},
		}},
		time.Second * 3600, true,
	}, "malformed header": {
		&http.Response{Header: map[string][]string{
			"Retry-After": []string{"malformed"},
		}},
		0, false,
	}, "missing header": {
		&http.Response{Header: map[string][]string{}},
		0, false,
	}}

	for name, test := range table {
		t.Logf("testing: %s", name)

		got, ok := GetRetryAfter(test.resp)
		if !reflect.DeepEqual(ok, test.ok) {
			t.Errorf("got: %t want: %t", ok, test.ok)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %v want: %v", got, test.want)
		}
	}
}
