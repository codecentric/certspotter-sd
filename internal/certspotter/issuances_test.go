package certspotter

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func mustParseTime(str string) time.Time {
	time, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return time
}

func TestClientGetIssuances(t *testing.T) {
	table := map[string]struct {
		data string
		opts *GetIssuancesOptions
		want []*Issuance
	}{"no options": {
		`[{
		   "id":"648494876",
		   "tbs_sha256":"b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
		   "pubkey_sha256":"8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
		   "not_before":"2018-11-28T00:00:00-00:00",
		   "not_after":"2020-12-02T12:00:00-00:00"
	     }]
        `,
		&GetIssuancesOptions{Domain: "example.com"},
		[]*Issuance{&Issuance{
			ID:           "648494876",
			TBSSHA256:    "b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
			PubKeySHA256: "8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
			NotBefore:    mustParseTime("2018-11-28T00:00:00-00:00"),
			NotAfter:     mustParseTime("2020-12-02T12:00:00-00:00"),
		}},
	}, "expand cert": {
		`[{
		   "id":"648494876",
		   "tbs_sha256":"b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
		   "pubkey_sha256":"8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
		   "not_before":"2018-11-28T00:00:00-00:00",
		   "not_after":"2020-12-02T12:00:00-00:00",
		   "cert":{
             "type":"cert",
             "sha256":"9250711c54de546f4370e0c3d3a3ec45bc96092a25a4a71a1afa396af7047eb8",
             "data":"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"
           }
	     }]
        `,
		&GetIssuancesOptions{
			Domain: "example.com",
			Expand: []string{"cert"},
		},
		[]*Issuance{&Issuance{
			ID:           "648494876",
			TBSSHA256:    "b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
			PubKeySHA256: "8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
			NotBefore:    mustParseTime("2018-11-28T00:00:00-00:00"),
			NotAfter:     mustParseTime("2020-12-02T12:00:00-00:00"),
			Certificate: &Certificate{
				Type:   "cert",
				SHA256: "9250711c54de546f4370e0c3d3a3ec45bc96092a25a4a71a1afa396af7047eb8",
				Data:   "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",
			},
		}},
	}, "expand dns_names": {
		`[{
		   "id":"648494876",
		   "tbs_sha256":"b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
           "dns_names": [
              "example.com", 
              "example.edu",
              "example.net",
              "example.org",
              "www.example.com",
              "www.example.edu",
              "www.example.net",
              "www.example.org"
           ],
		   "pubkey_sha256":"8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
		   "not_before":"2018-11-28T00:00:00-00:00",
		   "not_after":"2020-12-02T12:00:00-00:00"
	     }]
        `,
		&GetIssuancesOptions{
			Domain: "example.com",
			Expand: []string{"dns_names"},
		},
		[]*Issuance{&Issuance{
			ID:           "648494876",
			TBSSHA256:    "b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
			PubKeySHA256: "8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
			NotBefore:    mustParseTime("2018-11-28T00:00:00-00:00"),
			NotAfter:     mustParseTime("2020-12-02T12:00:00-00:00"),
			DNSNames: []string{
				"example.com",
				"example.edu",
				"example.net",
				"example.org",
				"www.example.com",
				"www.example.edu",
				"www.example.net",
				"www.example.org",
			},
		}},
	}, "expand issuer": {
		`[{
		   "id":"648494876",
		   "tbs_sha256":"b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
		   "pubkey_sha256":"8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
		   "issuer":{
             "name":"C=US, O=DigiCert Inc, CN=DigiCert SHA2 Secure Server CA",
             "pubkey_sha256":"e6426f344330d0a8eb080bbb7976391d976fc824b5dc16c0d15246d5148ff75c"
           },
		   "not_before":"2018-11-28T00:00:00-00:00",
		   "not_after":"2020-12-02T12:00:00-00:00"
	     }]
        `,
		&GetIssuancesOptions{
			Domain: "example.com",
			Expand: []string{"issuer"},
		},
		[]*Issuance{&Issuance{
			ID:           "648494876",
			TBSSHA256:    "b0537995114358761f330303e5b8a0d7c7319a7e458495395e07004911f91c38",
			PubKeySHA256: "8bd1da95272f7fa4ffb24137fc0ed03aae67e5c4d8b3c50734e1050a7920b922",
			NotBefore:    mustParseTime("2018-11-28T00:00:00-00:00"),
			NotAfter:     mustParseTime("2020-12-02T12:00:00-00:00"),
			Issuer: &Issuer{
				Name:         "C=US, O=DigiCert Inc, CN=DigiCert SHA2 Secure Server CA",
				PubKeySHA256: "e6426f344330d0a8eb080bbb7976391d976fc824b5dc16c0d15246d5148ff75c",
			},
		}},
	}}

	ctx := context.Background()
	cl, mux, stop := setup()
	defer stop()

	var tname string
	mux.HandleFunc("/issuances", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, table[tname].data)
	})

	for name, test := range table {
		t.Logf("testing: %s", name)

		tname = name
		got, _, err := cl.GetIssuances(ctx, test.opts)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("got: %v want %v", got, test.want)
		}
	}
}
