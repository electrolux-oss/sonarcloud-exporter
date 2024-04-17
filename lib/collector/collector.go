package collector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/jainlokesh2/sonarcloud-exporter/lib/client"
)

type Collector struct {
	up              *prometheus.Desc
	client          *client.ExporterClient
	projectInfo     *prometheus.Desc
	linesOfCode     *prometheus.Desc
	codeCoverage    *prometheus.Desc
	vulnerabilities *prometheus.Desc
	bugs            *prometheus.Desc
	codeSmells      *prometheus.Desc
	qualityGate     *prometheus.Desc
	enabledMetrics  map[string]bool
}

func New(c *client.ExporterClient, metricNames ...string) *Collector {
	log.Info("Creating collector")
	enabledMetrics := make(map[string]bool)
	allMetrics := []string{"up", "projectInfo", "linesOfCode", "codeCoverage", "vulnerabilities", "bugs", "codeSmells", "qualityGate"}

	if len(metricNames) == 1 && metricNames[0] == "all" {
		metricNames = allMetrics
	}

	for _, name := range metricNames {
		enabledMetrics[name] = true
	}

	return &Collector{
		up:              prometheus.NewDesc("sonarcloud_up", "Whether Sonarcloud scrape was successful", nil, nil),
		client:          c,
		projectInfo:     prometheus.NewDesc("sonarcloud_project_info", "General information about projects", []string{"project_name", "project_qualifier", "project_key", "project_organization"}, nil),
		linesOfCode:     prometheus.NewDesc("sonarcloud_lines_of_code", "Lines of code within a project in SonarCloud", []string{"project_key"}, nil),
		codeCoverage:    prometheus.NewDesc("sonarcloud_code_coverage", "Code coverage within a project in SonarCloud", []string{"project_key"}, nil),
		vulnerabilities: prometheus.NewDesc("sonarcloud_vulnerabilities", "Amount of vulnerabilities within a project in SonarCloud", []string{"project_key"}, nil),
		bugs:            prometheus.NewDesc("sonarcloud_bugs", "Amount of bugs within a project in SonarCloud", []string{"project_key"}, nil),
		codeSmells:      prometheus.NewDesc("sonarcloud_code_smells", "Amount of code smells within a project in SonarCloud", []string{"project_key"}, nil),
		qualityGate:     prometheus.NewDesc("sonarcloud_quality_gate", "Quality gate status of a project in SonarCloud", []string{"service"}, nil),
		enabledMetrics:  enabledMetrics,
	}
}
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up

	ch <- c.projectInfo

	ch <- c.linesOfCode
	ch <- c.codeCoverage
	ch <- c.bugs
	ch <- c.vulnerabilities
	ch <- c.codeSmells
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	log.Info("Running scrape")

	if stats, err := c.client.GetStats(); err != nil {
		log.Error(err)
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)

	} else {
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

		if c.enabledMetrics["projectInfo"] {
			collectProjectInfo(c, ch, stats)
		}

		collectMeasurements(c, ch, stats)

		if c.enabledMetrics["qualityGate"] {
			collectQualityGate(c, ch, stats)
		}

		log.Info("Scrape Complete")
	}
}

// Assume collectProjectInfo, collectMeasurements, and collectQualityGate functions check c.enabledMetrics before collecting each metric

func collectProjectInfo(c *Collector, ch chan<- prometheus.Metric, stats *client.Stats) {
	for _, project := range *stats.Projects {
		ch <- prometheus.MustNewConstMetric(c.projectInfo, prometheus.GaugeValue, 1, project.Name, project.Qualifier, project.Key, project.Organization)
	}
}

func collectMeasurements(c *Collector, ch chan<- prometheus.Metric, stats *client.Stats) {
	for _, measurement := range *stats.Measurements {
		value, err := strconv.ParseFloat(measurement.Value, 64)
		if err != nil {
			log.Error(err)
		}
		switch {
		case measurement.Metric == "ncloc" && c.enabledMetrics["linesOfCode"]:
			ch <- prometheus.MustNewConstMetric(c.linesOfCode, prometheus.GaugeValue, value, measurement.Key)
		case measurement.Metric == "coverage" && c.enabledMetrics["codeCoverage"]:
			ch <- prometheus.MustNewConstMetric(c.codeCoverage, prometheus.GaugeValue, value, measurement.Key)
		case measurement.Metric == "vulnerabilities" && c.enabledMetrics["vulnerabilities"]:
			ch <- prometheus.MustNewConstMetric(c.vulnerabilities, prometheus.GaugeValue, value, measurement.Key)
		case measurement.Metric == "bugs" && c.enabledMetrics["bugs"]:
			ch <- prometheus.MustNewConstMetric(c.bugs, prometheus.GaugeValue, value, measurement.Key)
		case measurement.Metric == "violations" && c.enabledMetrics["codeSmells"]:
			ch <- prometheus.MustNewConstMetric(c.codeSmells, prometheus.GaugeValue, value, measurement.Key)
		}
	}
}

func collectQualityGate(c *Collector, ch chan<- prometheus.Metric, stats *client.Stats) {

	processedMetrics := make(map[string]bool)
	for _, measurement := range *stats.QualityGate {
		organization := measurement.Organization + "_"
		trimmedKey := strings.TrimPrefix(measurement.Key, organization)
		metricKey := fmt.Sprintf("%s:%s", c.qualityGate, trimmedKey)
		if processedMetrics[metricKey] {
			continue
		}
		value, err := strconv.ParseFloat(measurement.Value, 64)
		if err != nil {
			log.Errorf("Error parsing metric value: %v", err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(c.qualityGate, prometheus.GaugeValue, value, trimmedKey)
		processedMetrics[metricKey] = true
	}
}
