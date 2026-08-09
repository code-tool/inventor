package metrics

import (
	"context"
	"testing"
	"time"

	"inventor/src/handler"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/redis/go-redis/v9"
)

func collect(cc SDTargetsCollector) []prometheus.Metric {
	ch := make(chan prometheus.Metric, 100)
	go func() {
		cc.Collect(ch)
		close(ch)
	}()
	var out []prometheus.Metric
	for m := range ch {
		out = append(out, m)
	}
	return out
}

func TestCollect_EmitsOneMetricPerTargetAddress(t *testing.T) {
	mr := miniredis.RunT(t)
	con := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sdTargets := &handler.SDTargets{Items: make(map[uuid.UUID]handler.StaticConfig)}
	ctx := context.Background()

	if _, err := sdTargets.Insert(handler.StaticConfig{
		Targets: []string{"10.0.0.1:9100", "10.0.0.2:9100"},
	}, ctx, con, 60); err != nil {
		t.Fatalf("unexpected error seeding target: %v", err)
	}

	cc := SDTargetsCollector{TargetInfo: &handler.SDTargetsMiddleware{
		SDTargets: sdTargets,
		Context:   ctx,
		Client:    con,
	}}

	metrics := collect(cc)
	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics (one per target address), got %d", len(metrics))
	}
	for _, m := range metrics {
		var out dto.Metric
		if err := m.Write(&out); err != nil {
			t.Fatalf("unexpected error writing metric: %v", err)
		}
	}
}

func TestCollect_EmitsInvalidMetricOnScanFailure(t *testing.T) {
	// Port 1 has nothing listening, so the client fails to reach Redis
	// and SDTargets.Scan returns an error.
	con := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond})
	cc := SDTargetsCollector{TargetInfo: &handler.SDTargetsMiddleware{
		SDTargets: &handler.SDTargets{Items: make(map[uuid.UUID]handler.StaticConfig)},
		Context:   context.Background(),
		Client:    con,
	}}

	metrics := collect(cc)
	if len(metrics) != 1 {
		t.Fatalf("expected exactly 1 invalid metric on scan failure, got %d", len(metrics))
	}
	var out dto.Metric
	if err := metrics[0].Write(&out); err == nil {
		t.Fatalf("expected the invalid metric's Write to return an error, got nil")
	}
}
