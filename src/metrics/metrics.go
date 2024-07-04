package metrics

import (
	"fmt"
	"inventor/src/handler"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ID                   = "id"
	Address              = "address"
	osReleaseMetricValue = float64(1)
	osReleseMetricDesc   = prometheus.NewDesc(
		"inventor_sd_target_info",
		"count of registered sd targets",
		[]string{
			ID,
			Address,
		}, nil,
	)
)

type SDTargetsCollector struct {
	TargetInfo *handler.SDTargetsMiddleware
}

func (cc SDTargetsCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(cc, ch)
}

func (cc SDTargetsCollector) Collect(ch chan<- prometheus.Metric) {
	rels, _ := cc.TargetInfo.SDTargets.Scan(cc.TargetInfo.Context, cc.TargetInfo.Client)
	for id, rel := range rels.Items {
		for _, target := range rel.Targets {
			ch <- prometheus.MustNewConstMetric(
				osReleseMetricDesc,
				prometheus.CounterValue,
				osReleaseMetricValue,
				fmt.Sprint(id),
				fmt.Sprint(target),
			)
		}

	}
}
