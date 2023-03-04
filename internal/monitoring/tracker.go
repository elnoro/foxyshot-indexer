package monitoring

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type Tracker struct {
	searchCounter prometheus.Counter
	indexCounter  prometheus.Counter
}

func NewTracker() *Tracker {
	return &Tracker{
		searchCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "search_request_count",
				Help: "No of requests handled by search handler",
			},
		),
		indexCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "index_image_count",
				Help: "No of images indexed",
			},
		),
	}
}

func (t *Tracker) Register() error {
	err := prometheus.Register(t.searchCounter)
	if err != nil {
		return fmt.Errorf("registering search counter, %w", err)
	}

	err = prometheus.Register(t.indexCounter)
	if err != nil {
		return fmt.Errorf("registering index counter, %w", err)
	}

	return nil
}

func (t *Tracker) OnIndex() {
	t.indexCounter.Inc()
}

func (t *Tracker) OnSearch() {
	t.searchCounter.Inc()
}
