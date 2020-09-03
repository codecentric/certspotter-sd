package export

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/codecentric/certspotter-sd/internal/certspotter"
	"github.com/codecentric/certspotter-sd/internal/config"
)

// Exporter is used for exporting issuances as targets to file.
type Exporter struct {
	logger *zap.SugaredLogger
}

// NewExporter returns a new exporter form global configuration.
func NewExporter(logger *zap.Logger, cfg *config.GlobalConfig) *Exporter {
	return &Exporter{
		logger: logger.Sugar(),
	}
}

// Export exports valid issuances from channel as targets to files
func (e *Exporter) Export(ctx context.Context, in <-chan []*certspotter.Issuance, cfgs []*config.FileConfig) {
	var issuances []*certspotter.Issuance
	for {
		select {
		case issuances = <-in:
			tgs := GetTargets(issuances)
			for filename, tgs := range GetFileTargets(tgs, cfgs) {
				if err := Write(filename, tgs); err != nil {
					e.logger.Errorw("writing targets to file", "err", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetTargets returns a set of valid targtes from issuances
func GetTargets(issuances []*certspotter.Issuance) []*Target {
	now := time.Now()
	var tgs []*Target
	for _, issuance := range issuances {
		if issuance.Certificate != nil && issuance.Certificate.Type == "precert" {
			continue
		}
		if now.After(issuance.NotAfter) || now.Before(issuance.NotBefore) {
			continue
		}
		tgs = append(tgs, NewTarget(issuance))
	}
	return tgs
}

// GetFileTargets returns a map of targets per matching file
func GetFileTargets(tgs []*Target, cfgs []*config.FileConfig) map[string][]*Target {
	files := make(map[string][]*Target)
	for _, cfg := range cfgs {
		files[cfg.File] = []*Target{}
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
func Write(filename string, tgs []*Target) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if tgs == nil {
		tgs = []*Target{}
	}
	return json.NewEncoder(file).Encode(tgs)
}
