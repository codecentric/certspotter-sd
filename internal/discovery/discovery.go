package discovery

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
	"github.com/codecentric/certspotter-sd/internal/config"
	"github.com/codecentric/certspotter-sd/internal/discovery/client"
	"github.com/codecentric/certspotter-sd/internal/version"
)

// Discovery is for subscribing to issuances for domains.
type Discovery struct {
	client *client.Client
	logger *zap.SugaredLogger
}

// NewDiscovery returns a new domain subscriber.
func NewDiscovery(logger *zap.Logger, cfg *config.GlobalConfig) *Discovery {
	return &Discovery{
		client: client.NewClient(logger, &client.Config{
			Interval:  cfg.Interval,
			RateLimit: cfg.RateLimit,
			Token:     cfg.Token,
			UserAgent: version.UserAgent(),
		}),
		logger: logger.Sugar(),
	}
}

// Discover subscribes to domains from configurations and sends all issuances to a channel on updates.
func (d *Discovery) Discover(ctx context.Context, cfgs []*config.DomainConfig) <-chan []*certspotter.Issuance {
	var chans []<-chan []*certspotter.Issuance
	for _, cfg := range cfgs {
		d.logger.Infow("subscribing to issuances", "domain", cfg.Domain)
		in := d.client.SubIssuances(ctx, &certspotter.GetIssuancesOptions{
			Domain:            cfg.Domain,
			Expand:            []string{"cert", "dns_names", "issuer"},
			IncludeSubdomains: cfg.IncludeSubdomains,
		})
		chans = append(chans, in)
	}

	ch := d.Merge(ctx, chans...)
	return d.Aggregate(ctx, ch)
}

// Aggregate aggregates issuances recived on in channel and outputs all previously recived issuances to a channel on updates
func (d *Discovery) Aggregate(ctx context.Context, in <-chan []*certspotter.Issuance) <-chan []*certspotter.Issuance {
	var all []*certspotter.Issuance
	var send chan []*certspotter.Issuance

	issuances := make(map[string]*certspotter.Issuance)
	out := make(chan []*certspotter.Issuance)
	go func() {
		defer close(out)
		for {
			select {
			case recieved := <-in:
				for _, issuance := range recieved {
					issuances[issuance.ID] = issuance
				}

				all = []*certspotter.Issuance{}
				for _, issuance := range issuances {
					all = append(all, issuance)
				}
				send = out
			case send <- all:
				send = nil
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Merge merges mulptiple issuance channels into one
func (d *Discovery) Merge(ctx context.Context, cs ...<-chan []*certspotter.Issuance) <-chan []*certspotter.Issuance {
	var wg sync.WaitGroup
	out := make(chan []*certspotter.Issuance)

	wg.Add(len(cs))
	for _, in := range cs {
		go func(in <-chan []*certspotter.Issuance) {
			defer wg.Done()
			for issuances := range in {
				select {
				case out <- issuances:
				case <-ctx.Done():
					return
				}
			}
		}(in)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
