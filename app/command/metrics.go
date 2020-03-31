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
		Name:      "synced_total",
		Help:      "The total number of consul keys synced",
		ConstLabels: prometheus.Labels{
			"source":   "git",
			"sink":     "consul",
			"state":    "success",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	consulGitSyncedFailed = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "synced_total",
		Help:      "The total number of consul keys synced",
		ConstLabels: prometheus.Labels{
			"source":   "git",
			"sink":     "consul",
			"state":    "failed",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	consulGitConnectionFailed = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "consul_connections_total",
		Help:      "The total number connections to consul",
		ConstLabels: prometheus.Labels{
			"source":   "git",
			"sink":     "consul",
			"state":    "failed",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	consulGitReads = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "git2consul",
		Name:      "git_reads_total",
		Help:      "The total number of times git was pulled",
		ConstLabels: prometheus.Labels{
			"source":   "git",
			"sink":     "consul",
			"instance": os.Getenv("HOSTNAME"),
		},
	})

	registry = prometheus.NewRegistry()
)

func init() {
	if os.Getenv("GIT2CONSUL_METRICS") == "true" {
		registry.MustRegister(consulGitSynced, consulGitSyncedFailed, consulGitConnectionFailed, consulGitReads)
	}
}

func pushMetrics(address string) {
	pusher := push.New("git2consul", address).Gatherer(registry)
	if err := pusher.Add(); err != nil {
		logrus.WithField("error", err).Warning("could not push to pushgateway")
	}
}
