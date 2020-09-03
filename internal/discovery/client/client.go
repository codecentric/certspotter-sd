package client

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
)

// Client is a thin wrapper around certspotter.Client.
type Client struct {
	client   *certspotter.Client
	interval time.Duration
	limiter  *rate.Limiter
	logger   *zap.SugaredLogger
}

// Config is used for configuring the client.
type Config struct {
	// Interval used between polling for new issuances.
	Interval time.Duration
	// RateLimit used for sending certspotter api requests in Hz.
	RateLimit float64
	// Token used for certspotter api.
	Token string
	// UserAgent used for client agent header.
	UserAgent string
}

// NewClient returns a new client for configuration.
func NewClient(logger *zap.Logger, cfg *Config) *Client {
	client := certspotter.NewClient(&certspotter.Config{
		Token:     cfg.Token,
		UserAgent: cfg.UserAgent,
	})
	limit := rate.Limit(cfg.RateLimit)
	limiter := rate.NewLimiter(limit, 5)

	return &Client{
		client:   client,
		interval: cfg.Interval,
		limiter:  limiter,
		logger:   logger.Sugar(),
	}
}

// GetIssuances returns issuances for options.
// It takes care of rate limiting and pagination.
func (c *Client) GetIssuances(ctx context.Context, opts *certspotter.GetIssuancesOptions) ([]*certspotter.Issuance, *http.Response, error) {
	var all []*certspotter.Issuance

	for {
		c.limiter.Wait(ctx)

		issuances, resp, err := c.client.GetIssuances(ctx, opts)
		if err != nil {
			return nil, nil, err
		}
		all = append(all, issuances...)

		if len(issuances) == 0 {
			return all, resp, nil
		}
		opts.After = issuances[len(issuances)-1].ID
	}
}

// SubIssuances returns a channel of issuances by subscribing to issuances for options.
func (c *Client) SubIssuances(ctx context.Context, opts *certspotter.GetIssuancesOptions) <-chan []*certspotter.Issuance {
	var delay time.Duration
	var ok bool

	ch := make(chan []*certspotter.Issuance)
	go func() {
		defer close(ch)
		for {
			select {
			case <-time.After(delay):
				issuances, resp, err := c.GetIssuances(ctx, opts)
				c.logger.Debugw("getting issuances", "issuances", len(issuances))
				if err != nil {
					c.logger.Errorw("getting issuances", "err", err)
				}

				delay, ok = GetRetryAfter(resp)
				if !ok || delay < c.interval {
					delay = c.interval
				}

				select {
				case ch <- issuances:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// GetRetryAfter returns Retry-After duration or false if non could be parsed.
func GetRetryAfter(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}

	header := resp.Header.Get("Retry-After")
	after, err := strconv.Atoi(header)

	if err != nil {
		return 0, false
	}
	return time.Second * time.Duration(after), true
}
