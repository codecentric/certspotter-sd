package export

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
)

func TestNewTarget(t *testing.T) {
	table := map[string]struct {
		issuance *certspotter.Issuance
		want     *Target
	}{"empty issuance": {
		&certspotter.Issuance{},
		&Target{Labels: map[string]string{
			"__meta_certspotter_id": "",
		}},
	}, "only cert sha256": {
		&certspotter.Issuance{
			ID: "648494876",
			Certificate: &certspotter.Certificate{
				SHA256: "9250711c54de546f4370e0c3d3a3ec45bc96092a25a4a71a1afa396af7047eb8",
			},
		},
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_id":          "648494876",
				"__meta_certspotter_cert_sha256": "9250711c54de546f4370e0c3d3a3ec45bc96092a25a4a71a1afa396af7047eb8",
			},
		},
	}, "only dns names": {
		&certspotter.Issuance{
			ID:       "648494876",
			DNSNames: []string{"example.com", "example2.com"},
		},
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_id":        "648494876",
				"__meta_certspotter_dns_names": "example.com;example2.com",
			},
			Targets: []string{"example.com", "example2.com"},
		},
	}, "only issuer name": {
		&certspotter.Issuance{
			ID: "648494876",
			Issuer: &certspotter.Issuer{
				Name: "C=US, O=DigiCert Inc, CN=DigiCert SHA2 Secure Server CA",
			},
		},
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_id":          "648494876",
				"__meta_certspotter_issuer_name": "C=US, O=DigiCert Inc, CN=DigiCert SHA2 Secure Server CA",
			},
		},
	}, "complete issuance": {
		&certspotter.Issuance{
			ID: "648494876",
			Certificate: &certspotter.Certificate{
				SHA256: "9250711c54de546f4370e0c3d3a3ec45bc96092a25a4a71a1afa396af7047eb8",
			},
			DNSNames: []string{"example.com", "example2.com"},
			Issuer: &certspotter.Issuer{
				Name: "C=US, O=DigiCert Inc, CN=DigiCert SHA2 Secure Server CA",
			},
		},
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_id":          "648494876",
				"__meta_certspotter_cert_sha256": "9250711c54de546f4370e0c3d3a3ec45bc96092a25a4a71a1afa396af7047eb8",
				"__meta_certspotter_dns_names":   "example.com;example2.com",
				"__meta_certspotter_issuer_name": "C=US, O=DigiCert Inc, CN=DigiCert SHA2 Secure Server CA",
			},
			Targets: []string{"example.com", "example2.com"},
		},
	}}

	for name, test := range table {
		t.Logf("testing: %s", name)

		got := NewTarget(test.issuance)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %#v want: %#v", got, test.want)
		}
	}
}

func TestTargetAddLabels(t *testing.T) {
	table := map[string]struct {
		target *Target
		labels map[string]string
		want   *Target
	}{"new labels": {
		&Target{Labels: make(map[string]string)},
		map[string]string{
			"name1": "val1",
			"name2": "val2",
		},
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_labels_name1": "val1",
				"__meta_certspotter_labels_name2": "val2",
			},
		},
	}, "existing labels": {
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_labels_name1": "val1",
				"__meta_certspotter_labels_name2": "val2",
			},
		},
		map[string]string{
			"name1": "val3",
			"name2": "val2",
		},
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_labels_name1": "val3",
				"__meta_certspotter_labels_name2": "val2",
			},
		},
	}}

	for name, test := range table {
		t.Logf("testing: %s", name)

		test.target.AddLabels(test.labels)
		got := test.target
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %#v want: %#v", got, test.want)
		}
	}
}

func TestTargetMatches(t *testing.T) {
	table := map[string]struct {
		target  *Target
		matches map[string]*regexp.Regexp
		want    bool
	}{"matching label": {
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_dns_names": "example.com",
			},
		},
		map[string]*regexp.Regexp{
			"dns_names": regexp.MustCompile("example.com"),
		},
		true,
	}, "non matching label": {
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_dns_names": "example.com",
			},
		},
		map[string]*regexp.Regexp{
			"dns_names": regexp.MustCompile("not-example.com"),
		},
		false,
	}, "non prefixed label": {
		&Target{
			Labels: map[string]string{
				"dns_names": "example.com",
			},
		},
		map[string]*regexp.Regexp{
			"dns_names": regexp.MustCompile("example.com"),
		},
		false,
	}, "non matches": {
		&Target{
			Labels: map[string]string{
				"__meta_certspotter_dns_names": "example.com",
			},
		},
		map[string]*regexp.Regexp{},
		true,
	}}

	for name, test := range table {
		t.Logf("testing: %s", name)

		got := test.target.Matches(test.matches)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %t want: %t", got, test.want)
		}
	}
}
