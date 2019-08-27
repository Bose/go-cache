package cache

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	subsystem = "go_cache"
)

func incrementCounterVec(c *prometheus.CounterVec, statusCode int) {
	if c != nil {
		c.WithLabelValues(fmt.Sprintf("%d", statusCode)).Inc()
	}
}

func initCounterVec(m *prometheus.CounterVec, name string, help string) *prometheus.CounterVec {
	if m == nil {
		m = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      name,
				Help:      help,
			},
			[]string{"status_code"},
		)
		if err := prometheus.Register(m); err != nil {
			logrus.Infof("%s could not be registered: " + name + err.Error())

		} else {
			logrus.Infof("%s registered.", name)
		}
	}
	return m
}
func initSummary(m prometheus.Summary, name string, help string) prometheus.Summary {
	if m == nil {
		m = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      name,
				Help:      help,
			},
		)
		if err := prometheus.Register(m); err != nil {
			logrus.Infof("%s could not be registered: " + name + err.Error())
		} else {
			logrus.Infof("%s registered.", name)
		}
	}
	return m
}

func observeSummary(s prometheus.Summary, start time.Time) {
	elapsed := float64(time.Since(start)) / float64(time.Second)
	s.Observe(elapsed)
}

func initGaugeWithFunc(gaugeFunc func() float64, name string, help string) error {
	gf := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		gaugeFunc,
	)
	if err := prometheus.Register(gf); err != nil {
		logrus.Infof("%s could not be registered: %s", name, err.Error())
		return err
	}
	logrus.Infof("%s registered.", name)
	return nil
}
