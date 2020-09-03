package config

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	yaml "gopkg.in/yaml.v2"
)

var (
	regexDomainName = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
)

var (
	// DefaultConfig is the top-level configuration.
	DefaultConfig = Config{
		GlobalConfig: DefaultGlobalConfig,
	}

	// DefaultGlobalConfig is the default global configuration.
	DefaultGlobalConfig = GlobalConfig{
		Interval:  time.Hour,
		RateLimit: 1.25,
	}

	// DefaultDomainConfig is the default domain configuration.
	DefaultDomainConfig = DomainConfig{
		IncludeSubdomains: false,
	}
)

// Config is the top-level configuration.
type Config struct {
	GlobalConfig  GlobalConfig    `yaml:"global"`
	DomainConfigs []*DomainConfig `yaml:"domains"`
	FileConfigs   []*FileConfig   `yaml:"files"`
}

// GlobalConfig configures globally shared values.
type GlobalConfig struct {
	// Interval to use between polling the certspotter api.
	Interval time.Duration `yaml:"polling_interval"`
	// RateLimit to use for certspotter api (configured in Hz).
	RateLimit float64 `yaml:"rate_limit"`
	// Token to used for authenticating againts certspotter api.
	Token string `yaml:"token"`
}

// DomainConfig configures domain requesting options.
type DomainConfig struct {
	// Domain to use for requesting certificate issuances.
	Domain string `yaml:"domain"`
	// If sub domains should be included.
	IncludeSubdomains bool `yaml:"include_subdomains"`
}

// FileConfig configure a file for exporting issuances.
type FileConfig struct {
	// Filename to export targets to
	File string `yaml:"file"`
	// Labels to add to targets before export
	Labels map[string]string `yaml:"labels"`
	// Matches for target to be included in file
	MatchRE MatchRE `yaml:"match_re"`
}

// MatchRE represents a map of regex patterns
type MatchRE map[string]*regexp.Regexp

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *GlobalConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultGlobalConfig
	type plain GlobalConfig

	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if c.Interval <= 0 {
		return fmt.Errorf("polling interval %s must be greater than 0s", c.Interval)
	}
	if c.RateLimit <= 0 {
		return fmt.Errorf("rate limit %fHz must be greater than 0Hz", c.RateLimit)
	}
	if c.RateLimit > 20 {
		return fmt.Errorf("rate limit %fHz must be smaller than 20Hz", c.RateLimit)
	}

	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *DomainConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultDomainConfig
	type plain DomainConfig

	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if !regexDomainName.MatchString(c.Domain) {
		return fmt.Errorf("domain %s must be a valid domain", c.Domain)
	}

	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (m *MatchRE) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var matches map[string]string
	if err := unmarshal(&matches); err != nil {
		return err
	}

	mm := make(map[string]*regexp.Regexp, len(matches))
	for name, str := range matches {
		regex, err := regexp.Compile("^" + str + "$")
		if err != nil {
			return err
		}
		mm[name] = regex
	}
	*m = mm

	return nil
}

// Load parses the YAML input s into a Config.
func Load(data string) (*Config, error) {
	cfg := &Config{}
	*cfg = DefaultConfig

	err := yaml.UnmarshalStrict([]byte(data), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadFile parses the given YAML file into a Config.
func LoadFile(filename string) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg, err := Load(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing YAML file %s: %w", filename, err)
	}
	return cfg, nil
}
