package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/codecentric/certspotter-sd/internal/config"
	"github.com/codecentric/certspotter-sd/internal/discovery"
	"github.com/codecentric/certspotter-sd/internal/export"
	"github.com/codecentric/certspotter-sd/internal/version"
)

type arguments struct {
	ConfigFile string
	LogLevel   *zapcore.Level
}

func main() {
	args := argsparse()

	logger := getlogger(*args.LogLevel)
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg, err := config.LoadFile(args.ConfigFile)
	if err != nil {
		sugar.Fatalw("can't read configuration", "err", err)
	}

	dc := discovery.NewDiscovery(
		logger.With(zap.String("component", "discovery")),
		&cfg.GlobalConfig,
	)
	exporter := export.NewExporter(
		logger.With(zap.String("component", "exporter")),
		&cfg.GlobalConfig,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go sighandler(ctx, func(sig os.Signal) {
		sugar.Infow("stopping service discovery", "signal", sig)
		cancel()
		os.Exit(0)
	})

	sugar.Info("starting service discovery")
	ch := dc.Discover(ctx, cfg.DomainConfigs)
	exporter.Export(ctx, ch, cfg.FileConfigs)
}

func argsparse() *arguments {
	var fversion bool

	args := arguments{}
	flag.StringVar(&args.ConfigFile, "config.file",
		"/etc/prometheus/certspotter-sd.yml",
		"configuration file to use.",
	)
	args.LogLevel = zap.LevelFlag("log.level",
		zap.InfoLevel,
		"severity of log to write. (default info)",
	)
	flag.BoolVar(&fversion, "version",
		false,
		"print certspotter-sd version.",
	)
	flag.Parse()

	if fversion {
		fmt.Println(version.Print())
		os.Exit(0)
	}

	return &args
}

func getlogger(lvl zapcore.Level) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level.SetLevel(lvl)

	logger, err := cfg.Build()
	if err != nil {
		log.Fatalf("can't initialize logger: %v", err)
	}
	return logger
}

func sighandler(ctx context.Context, handler func(os.Signal)) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	for {
		select {
		case sig := <-ch:
			handler(sig)
		case <-ctx.Done():
			return
		}
	}
}
