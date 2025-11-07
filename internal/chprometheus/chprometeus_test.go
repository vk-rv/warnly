package chprometheus_test

import (
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vk-rv/warnly/internal/chprometheus"
)

// mockStatGetter implements StatGetter for testing.
type mockStatGetter struct {
	stats driver.Stats
}

func (m *mockStatGetter) Stats() driver.Stats {
	return m.stats
}

func TestClickhouseCollector(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewPedanticRegistry()
	{
		getter := &mockStatGetter{stats: driver.Stats{
			Idle:         5,
			Open:         10,
			MaxIdleConns: 20,
			MaxOpenConns: 50,
		}}
		if err := reg.Register(chprometheus.NewClickhouseCollector(getter, "ch_A")); err != nil {
			t.Fatal(err)
		}
	}
	{
		getter := &mockStatGetter{stats: driver.Stats{
			Idle:         3,
			Open:         8,
			MaxIdleConns: 15,
			MaxOpenConns: 40,
		}}
		if err := reg.Register(chprometheus.NewClickhouseCollector(getter, "ch_B")); err != nil {
			t.Fatal(err)
		}
	}

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	names := []string{
		"analytics_conn_idle_current",
		"analytics_conn_open_current",
		"analytics_conn_max_idle_current",
		"analytics_conn_max_open_current",
	}
	type result struct {
		found bool
	}
	results := make(map[string]result)
	for _, name := range names {
		results[name] = result{found: false}
	}
	for _, mf := range mfs {
		m := mf.GetMetric()
		if len(m) != 2 {
			t.Errorf("expected 2 metrics but got %d", len(m))
		}
		labelA := m[0].GetLabel()[0]
		if name := labelA.GetName(); name != "db_name" {
			t.Errorf("expected to get label \"db_name\" but got %s", name)
		}
		if value := labelA.GetValue(); value != "ch_A" {
			t.Errorf("expected to get value \"ch_A\" but got %s", value)
		}
		labelB := m[1].GetLabel()[0]
		if name := labelB.GetName(); name != "db_name" {
			t.Errorf("expected to get label \"db_name\" but got %s", name)
		}
		if value := labelB.GetValue(); value != "ch_B" {
			t.Errorf("expected to get value \"ch_B\" but got %s", value)
		}

		for _, name := range names {
			if name == mf.GetName() {
				results[name] = result{found: true}
				break
			}
		}
	}

	for name, result := range results {
		if !result.found {
			t.Errorf("%s not found", name)
		}
	}
}
