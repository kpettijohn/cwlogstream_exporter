package main

import (
	"net/http"
	"os"

	"github.com/kpettijohn/cwlogstream_exporter/collector"
	"github.com/kpettijohn/cwlogstream_exporter/internal/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

// Variables populated at build time.
// Example: go build -ldflags="-X main.exporterVersion=0.0.1"
var (
	exporterBranch   string
	exporterVersion  string
	exporterRevision string
)

func main() {
	log.Info("Starting AWS Log Stream exporter...")

	// Parse command line flags
	if err := parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	// Enable/disable debug logging
	debug := os.Getenv("DEBUG")

	log.SetLevel("info")
	if len(debug) != 0 {
		log.SetLevel("debug")
	}

	if cfg.debug {
		log.SetLevel("debug")
	}
	version.Branch = exporterBranch
	version.Version = exporterVersion
	version.Revision = exporterRevision
	prometheus.MustRegister(version.NewCollector("cwlogstream"))

	// Create the exporter and register it
	exporter, err := collector.New(cfg.awsRegion, cfg.logGroupPrefix, cfg.logStreamTimeout, cfg.ec2InstanceTagFilter)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)

	// Serve metrics
	http.Handle(cfg.metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>AWS ClouwWatch Log Stream (cwlogstream) Exporter</title></head>
             <body>
             <h1>AWS Log Stream Exporter</h1>
             <p><a href='` + cfg.metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Info("Listening on", cfg.listenAddress)
	log.Fatal(http.ListenAndServe(cfg.listenAddress, nil))
}
