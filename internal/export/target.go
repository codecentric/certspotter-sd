package export

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
)

// Target represents a prometheus file service discovery target
type Target struct {
	Labels  map[string]string `json:"labels"`
	Targets []string          `json:"targets"`
}

// NewTarget returns a new target from a certspotter issuance.
func NewTarget(issuance *certspotter.Issuance) *Target {
	labels := make(map[string]string)

	labels["__meta_certspotter_id"] = issuance.ID
	if issuance.Certificate != nil {
		labels["__meta_certspotter_cert_sha256"] = issuance.Certificate.SHA256
	}
	if len(issuance.DNSNames) != 0 {
		labels["__meta_certspotter_dns_names"] = strings.Join(issuance.DNSNames, ";")
	}
	if issuance.Issuer != nil {
		labels["__meta_certspotter_issuer_name"] = issuance.Issuer.Name
	}

	var targets []string
	for _, name := range issuance.DNSNames {
		if !strings.HasPrefix(name, "*.") {
			targets = append(targets, name)
		}
	}

	return &Target{
		Labels:  labels,
		Targets: targets,
	}
}

// AddLabels adds labels to target with prefix __meta_certspotter_labels_
func (t *Target) AddLabels(labels map[string]string) {
	for name, val := range labels {
		label := fmt.Sprintf("__meta_certspotter_labels_%s", name)
		t.Labels[label] = val
	}
}

// Matches tests if target labels match map of regex patterns.
// __meta_certspotter_ is removed from target labels before matching.
func (t *Target) Matches(matches map[string]*regexp.Regexp) bool {
	for name, re := range matches {
		label := fmt.Sprintf("__meta_certspotter_%s", name)
		if val, ok := t.Labels[label]; !ok || !re.MatchString(val) {
			return false
		}
	}
	return true
}
