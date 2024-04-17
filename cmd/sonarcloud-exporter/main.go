package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/jainlokesh2/sonarcloud-exporter/internal"
	"github.com/jainlokesh2/sonarcloud-exporter/lib/client"
	"github.com/jainlokesh2/sonarcloud-exporter/lib/collector"
)

var config internal.Config

func init() {
	flag.StringVar(&config.Token, "scToken", os.Getenv("SC_TOKEN"), "Token to access SonarCloud API")
	flag.StringVar(&config.ListenAddress, "listenAddress", getDefaultEnv("LISTEN_ADDRESS", "8080"), "Port address of exporter to run on")
	flag.StringVar(&config.ListenPath, "listenPath", getDefaultEnv("LISTEN_PATH", "/metrics"), "Path where metrics will be exposed")
	flag.StringVar(&config.Organization, "organization", os.Getenv("SC_ORGANIZATION"), "Organization to query within SonarCloud")
	flag.StringVar(&config.MetricsName, "metricsName", getDefaultEnv("METRICS_NAME", "all"), "Comma-separated list of metrics to enable")
}

func getDefaultEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	flag.Parse()

	if config.Token == "" {
		log.Error("SonarCloud API token is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Info("Starting SonarCloud Exporter")

	client := client.New(config)
	metricNames := strings.Split(config.MetricsName, ",")
	coll := collector.New(client, metricNames...)

	prometheus.MustRegister(coll)

	log.Info("Start serving metrics")

	http.Handle(config.ListenPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
			<head><title>SonarCloud Exporter</title></head>
			<body>
			<h1>SonarCloud Exporter</h1>
			<p><a href="` + config.ListenPath + `">Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			log.Error(err)
		}
	})

	log.Fatal(http.ListenAndServe(":"+config.ListenAddress, nil))
}
