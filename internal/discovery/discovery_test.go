package discovery

import (
	"reflect"
	"testing"
	"time"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
	"github.com/codecentric/certspotter-sd/internal/discovery/target"
)

func mustParseTime(str string) time.Time {
	time, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return time
}

func TestGetTargets(t *testing.T) {
	table := map[string]struct {
		issuances []*certspotter.Issuance
		want      []*target.Target
	}{"valid issuances": {
		[]*certspotter.Issuance{
			&certspotter.Issuance{
				ID:          "648494876",
				NotBefore:   mustParseTime("2000-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2100-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "cert"},
			},
			&certspotter.Issuance{
				ID:          "648494877",
				NotBefore:   mustParseTime("2000-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2100-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "cert"},
			},
		},
		[]*target.Target{
			&target.Target{Labels: map[string]string{
				"__meta_certspotter_id":          "648494876",
				"__meta_certspotter_cert_sha256": "",
			}},
			&target.Target{Labels: map[string]string{
				"__meta_certspotter_id":          "648494877",
				"__meta_certspotter_cert_sha256": "",
			}},
		},
	}, "precert issuances": {
		[]*certspotter.Issuance{
			&certspotter.Issuance{
				ID:          "648494876",
				NotBefore:   mustParseTime("2000-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2100-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "cert"},
			},
			&certspotter.Issuance{
				ID:          "648494877",
				NotBefore:   mustParseTime("2000-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2100-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "precert"},
			},
		},
		[]*target.Target{
			&target.Target{Labels: map[string]string{
				"__meta_certspotter_id":          "648494876",
				"__meta_certspotter_cert_sha256": "",
			}},
		},
	}, "outdated issuances": {
		[]*certspotter.Issuance{
			&certspotter.Issuance{
				ID:          "648494876",
				NotBefore:   mustParseTime("2000-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2100-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "cert"},
			},
			&certspotter.Issuance{
				ID:          "648494877",
				NotBefore:   mustParseTime("2000-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2000-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "cert"},
			},
			&certspotter.Issuance{
				ID:          "648494877",
				NotBefore:   mustParseTime("2100-01-01T00:00:00-00:00"),
				NotAfter:    mustParseTime("2100-01-01T00:00:00-00:00"),
				Certificate: &certspotter.Certificate{Type: "cert"},
			},
		},
		[]*target.Target{
			&target.Target{Labels: map[string]string{
				"__meta_certspotter_id":          "648494876",
				"__meta_certspotter_cert_sha256": "",
			}},
		},
	}}

	for name, test := range table {
		t.Logf("testing: %s", name)

		got := GetTargets(test.issuances)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %+v want: %+v", got[0], test.want[0])
		}
	}
}
