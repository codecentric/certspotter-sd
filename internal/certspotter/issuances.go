package certspotter

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"
)

var _ sort.Interface = &Issuances{}

// Certificate represents a cerspotter certificate object.
type Certificate struct {
	Data   string `json:"data"`
	SHA256 string `json:"sha256"`
	Type   string `json:"type"`
}

// Issuance represents a cerspotter issuance object.
type Issuance struct {
	ID        string   `json:"id"`
	DNSNames  []string `json:"dns_names"`
	TBSSHA256 string   `json:"tbs_sha256"`

	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	PubKeySHA256 string    `json:"pubkey_sha256"`

	Issuer      *Issuer      `json:"issuer"`
	Certificate *Certificate `json:"cert"`
}

// Issuances implements sort.Interface.
type Issuances []*Issuance

// Issuer represents a cerspotter issuer object.
type Issuer struct {
	Name         string `json:"name"`
	PubKeySHA256 string `json:"pubkey_sha256"`
}

// GetIssuancesOptions are options used when getting issuances.
type GetIssuancesOptions struct {
	Domain            string   `url:"domain"`
	IncludeSubdomains bool     `url:"include_subdomains,omitempty"`
	MatchWildcards    bool     `url:"match_wildcards,omitempty"`
	After             string   `url:"after,omitempty"`
	Expand            []string `url:"expand,omitempty"`
}

// GetIssuances returns issuances and response for options.
func (c *Client) GetIssuances(ctx context.Context, opts *GetIssuancesOptions) ([]*Issuance, *http.Response, error) {
	var val []*Issuance
	resp, err := c.Do(ctx, &val, &DoOptions{
		Method:     "GET",
		Path:       "/issuances",
		Parameters: opts,
	})
	return val, resp, err
}

// Len, Swap, Less implement sort.Interface
func (is Issuances) Len() int           { return len(is) }
func (is Issuances) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }
func (is Issuances) Less(i, j int) bool { return strings.Compare(is[i].ID, is[j].ID) == -1 }
