package discovery

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
	"github.com/codecentric/certspotter-sd/internal/config"
	"github.com/codecentric/certspotter-sd/internal/discovery/client"
	"github.com/codecentric/certspotter-sd/internal/discovery/target"
	"github.com/codecentric/certspotter-sd/internal/version"
)

var (
	targetsDiscoveredMetric = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "certspotter_targets_discovered",
			Help: "The current number of targets from issuances",
		},
	)
	targetsWrittenMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "certspotter_targets_written",
			Help: "The current number of targets written to file",
		},
		[]string{"filename"},
	)
)

// Discovery is used for exporting issuances as targets to file.
type Discovery struct {
	client    *client.Client
	cfg       *config.Config
	issuances []*certspotter.Issuance
	logger    *zap.SugaredLogger
	mtx       sync.RWMutex
	send      chan struct{}
}

// NewDiscovery returns a new discovery form global configuration.
func NewDiscovery(logger *zap.Logger, cfg *config.Config) *Discovery {
	return &Discovery{
		cfg: cfg,
		client: client.NewClient(logger, &client.Config{
			Interval:  cfg.GlobalConfig.Interval,
			RateLimit: cfg.GlobalConfig.RateLimit,
			Token:     cfg.GlobalConfig.Token,
			UserAgent: version.UserAgent(),
		}),
		logger: logger.Sugar(),
	}
}

// Discover discovers prometheus targets from certificate issuances and writes
// all valif targets to files.
func (d *Discovery) Discover(ctx context.Context) {
	d.logger.Infow("starting discovering issuances")

	var chans []<-chan []*certspotter.Issuance
	for _, cfg := range d.cfg.DomainConfigs {
		d.logger.Infow("subscribing to issuances", "domain", cfg.Domain)
		in := d.client.SubIssuances(ctx, &certspotter.GetIssuancesOptions{
			Domain:            cfg.Domain,
			Expand:            []string{"cert", "dns_names", "issuer"},
			IncludeSubdomains: cfg.IncludeSubdomains,
		})
		chans = append(chans, in)
	}

	d.send = make(chan struct{})
	defer close(d.send)

	for _, ch := range chans {
		go d.collect(ctx, ch)
	}
	d.export(ctx)
}

// collect collects issuances from channel to internal structure.
func (d *Discovery) collect(ctx context.Context, ch <-chan []*certspotter.Issuance) {
	for {
		select {
		case issuances := <-ch:
			d.mtx.RLock()
			d.issuances = append(d.issuances, issuances...)
			d.mtx.RUnlock()
			d.send <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

// export writes issuances as targets to files.
func (d *Discovery) export(ctx context.Context) {
	for {
		select {
		case <-d.send:
			tgs := GetTargets(d.issuances)
			d.logger.Debugw("got targets from issuances",
				"targets", len(tgs),
				"issuances", len(d.issuances),
			)
			targetsDiscoveredMetric.Set(float64(len(tgs)))

			for filename, tgs := range GetFileTargets(tgs, d.cfg.FileConfigs) {
				d.logger.Debugw("writing targets to file",
					"filename", filename,
					"targets", len(tgs),
				)
				if err := Write(filename, tgs); err != nil {
					d.logger.Errorw("writing targets to file",
						"filename", filename,
						"err", err,
					)
				}
				targetsWrittenMetric.WithLabelValues(
					filename,
				).Set(float64(len(tgs)))
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetTargets returns a set of valid targtes from issuances
func GetTargets(issuances []*certspotter.Issuance) []*target.Target {
	now := time.Now()
	var tgs []*target.Target
	for _, issuance := range issuances {
		if now.After(issuance.NotAfter) || now.Before(issuance.NotBefore) {
			continue
		}
		tgs = append(tgs, target.NewTarget(issuance))
	}
	return tgs
}

// GetFileTargets returns a map of targets per matching file
func GetFileTargets(tgs []*target.Target, cfgs []*config.FileConfig) map[string][]*target.Target {
	files := make(map[string][]*target.Target)
	for _, cfg := range cfgs {
		files[cfg.File] = []*target.Target{}
	}

	for _, tg := range tgs {
		for _, cfg := range cfgs {
			if !tg.Matches(cfg.MatchRE) {
				continue
			}
			tg.AddLabels(cfg.Labels)
			files[cfg.File] = append(files[cfg.File], tg)
		}
	}
	return files
}

// Write writes targets as json array to filename
func Write(filename string, tgs []*target.Target) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if tgs == nil {
		tgs = []*target.Target{}
	}
	return json.NewEncoder(file).Encode(tgs)
}
