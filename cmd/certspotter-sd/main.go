package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/codecentric/certspotter-sd/internal/config"
	"github.com/codecentric/certspotter-sd/internal/discovery"
	"github.com/codecentric/certspotter-sd/internal/version"
)

type arguments struct {
	ConfigFile string
	LogLevel   *zapcore.Level
	MetricPort int
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

	discovery := discovery.NewDiscovery(
		logger.With(zap.String("component", "discovery")),
		cfg,
	)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", args.MetricPort), nil)

	ctx, cancel := context.WithCancel(context.Background())
	go sighandler(ctx, func(sig os.Signal) {
		sugar.Infow("stopping service discovery", "signal", sig)
		cancel()
		os.Exit(0)
	})
	discovery.Discover(ctx)
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
	flag.IntVar(&args.MetricPort, "metric.port",
		9800,
		"port to expose metrics to.",
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
