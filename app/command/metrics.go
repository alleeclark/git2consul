package command

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
)

var (
	consulGitSynced = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "synced_success",
		Help:      "The total number of consul keys synced",
		ConstLabels: prometheus.Labels{
			"state":    "success",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	consulGitSyncedFailed = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "synced_failed",
		Help:      "The total number of consul keys synced failed",
		ConstLabels: prometheus.Labels{
			"state":    "failed",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	consulGitConnectionFailed = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "consul_connection_failed",
		Help:      "The total number connections to consul failed",
		ConstLabels: prometheus.Labels{
			"state":    "failed",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	consulGitReads = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "git_reads",
		Help:      "The total number of times git was pulled",
		ConstLabels: prometheus.Labels{
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	registry = prometheus.NewRegistry()
)

func init() {
	if os.Getenv("GIT2CONSUL_METRICS") == "true" {
		registry.Register(consulGitSynced)
		registry.Register(consulGitSyncedFailed)
		registry.Register(consulGitConnectionFailed)
		registry.Register(consulGitReads)
	}
}

func pushMetrics(address string) {
	gauges := append([]prometheus.Gauge{}, consulGitReads, consulGitConnectionFailed, consulGitSyncedFailed, consulGitSynced)
	pusher := push.New("git2consul", address)
	for _, gauge := range gauges {
		if err := pusher.Collector(gauge).Gatherer(prometheus.DefaultGatherer).Push(); err != nil {
			logrus.Warningln(err)
		}
	}
}
