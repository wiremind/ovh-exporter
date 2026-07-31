package network

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// testScope is a throwaway Scope implementation used only to exercise
// RefreshScope's generic mechanism, independent of any real gauge's label
// keys (see ProjectScope/ServiceScope/GlobalScope in scope.go for the
// production ones).
type testScope prometheus.Labels

func (s testScope) Labels() prometheus.Labels { return prometheus.Labels(s) }

func TestRefreshScope(t *testing.T) {
	cases := map[string]struct {
		seed           map[[2]string]float64 // [scope, item] -> value, set on the gauge before RefreshScope runs
		scope          Scope
		itemScopeValue string   // "scope" label used by set for freshly fetched items, mirrors a real collector capturing projectID from its enclosing closure
		fetchItems     []string // items returned by the fetch; nil combined with fetchErr means "fetch failed"
		fetchErr       error
		wantErr        bool
		wantSeries     string // full expected Prometheus text exposition after RefreshScope runs
	}{
		"fetch succeeds: stale items in scope are cleared, fresh ones set": {
			seed: map[[2]string]float64{
				{"p1", "x"}: 1,
				{"p1", "y"}: 1,
				{"p2", "z"}: 1,
			},
			scope:          testScope{"scope": "p1"},
			itemScopeValue: "p1",
			fetchItems:     []string{"x2"},
			wantSeries: `
				# HELP test_gauge test gauge
				# TYPE test_gauge gauge
				test_gauge{item="x2",scope="p1"} 1
				test_gauge{item="z",scope="p2"} 1
			`,
		},
		"fetch fails: previous values in scope are kept untouched": {
			seed: map[[2]string]float64{
				{"p1", "x"}: 1,
				{"p2", "z"}: 1,
			},
			scope:    testScope{"scope": "p1"},
			fetchErr: errors.New("ovh api down"),
			wantErr:  true,
			wantSeries: `
				# HELP test_gauge test gauge
				# TYPE test_gauge gauge
				test_gauge{item="x",scope="p1"} 1
				test_gauge{item="z",scope="p2"} 1
			`,
		},
		"fetch succeeds with no items: scope is cleared and stays empty": {
			seed: map[[2]string]float64{
				{"p1", "x"}: 1,
				{"p2", "z"}: 1,
			},
			scope:      testScope{"scope": "p1"},
			fetchItems: nil,
			wantSeries: `
				# HELP test_gauge test gauge
				# TYPE test_gauge gauge
				test_gauge{item="z",scope="p2"} 1
			`,
		},
		"GlobalScope on success clears every series, like Reset": {
			seed: map[[2]string]float64{
				{"p1", "x"}: 1,
				{"p2", "z"}: 1,
			},
			scope:      GlobalScope{},
			fetchItems: []string{"w"},
			wantSeries: `
				# HELP test_gauge test gauge
				# TYPE test_gauge gauge
				test_gauge{item="w",scope=""} 1
			`,
		},
		"fetch fails with GlobalScope: nothing is reset, unlike the old Reset() call": {
			seed: map[[2]string]float64{
				{"p1", "x"}: 1,
				{"p2", "z"}: 1,
			},
			scope:    GlobalScope{},
			fetchErr: errors.New("ovh api down"),
			wantErr:  true,
			wantSeries: `
				# HELP test_gauge test gauge
				# TYPE test_gauge gauge
				test_gauge{item="x",scope="p1"} 1
				test_gauge{item="z",scope="p2"} 1
			`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			gauge := prometheus.NewGaugeVec(
				prometheus.GaugeOpts{Name: "test_gauge", Help: "test gauge"},
				[]string{"scope", "item"},
			)
			for key, value := range tc.seed {
				gauge.With(prometheus.Labels{"scope": key[0], "item": key[1]}).Set(value)
			}

			err := RefreshScope(
				tc.scope,
				"test_collector",
				func() ([]string, error) {
					if tc.fetchErr != nil {
						return nil, tc.fetchErr
					}
					return tc.fetchItems, nil
				},
				func(item string) {
					gauge.With(prometheus.Labels{"scope": tc.itemScopeValue, "item": item}).Set(1)
				},
				gauge,
			)

			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if err := testutil.CollectAndCompare(gauge, strings.NewReader(tc.wantSeries), "test_gauge"); err != nil {
				t.Fatalf("unexpected gauge state:\n%v", err)
			}
		})
	}
}

func TestRefreshScope_MultipleGaugesClearedTogether(t *testing.T) {
	subscription := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "test_subscription", Help: "test subscription"},
		[]string{"server_id"},
	)
	expiration := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "test_expiration", Help: "test expiration"},
		[]string{"server_id"},
	)
	subscription.With(prometheus.Labels{"server_id": "old"}).Set(1)
	expiration.With(prometheus.Labels{"server_id": "old"}).Set(123)

	err := RefreshScope(
		GlobalScope{},
		"test_collector",
		func() ([]string, error) { return []string{"new"}, nil },
		func(serverID string) {
			subscription.With(prometheus.Labels{"server_id": serverID}).Set(1)
			expiration.With(prometheus.Labels{"server_id": serverID}).Set(456)
		},
		subscription, expiration,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := testutil.CollectAndCompare(subscription, strings.NewReader(`
		# HELP test_subscription test subscription
		# TYPE test_subscription gauge
		test_subscription{server_id="new"} 1
	`), "test_subscription"); err != nil {
		t.Fatalf("unexpected subscription gauge state:\n%v", err)
	}
	if err := testutil.CollectAndCompare(expiration, strings.NewReader(`
		# HELP test_expiration test expiration
		# TYPE test_expiration gauge
		test_expiration{server_id="new"} 456
	`), "test_expiration"); err != nil {
		t.Fatalf("unexpected expiration gauge state:\n%v", err)
	}
}

// TestRefreshScope_FetchAndSetShareStateViaPointer mirrors the real pattern
// in serve_cloud_project_instance_billing.go: fetch makes a second,
// dependent API call (there: flavors, keyed off the instances fetch just
// returned) and shares that extra result with set through an explicit
// pointer, instead of an implicitly captured bare var. This pins down that
// set only ever observes the pointer AFTER fetch has written to it.
func TestRefreshScope_FetchAndSetShareStateViaPointer(t *testing.T) {
	enrichment := new(string)

	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "test_gauge_enrichment", Help: "h"},
		[]string{"item", "enrichment"},
	)

	err := RefreshScope(
		GlobalScope{},
		"test_collector",
		func() ([]string, error) {
			*enrichment = "fetched-once" // written here, before any item is set
			return []string{"a", "b"}, nil
		},
		func(item string) {
			gauge.With(prometheus.Labels{"item": item, "enrichment": *enrichment}).Set(1) // read here, for every item
		},
		gauge,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := testutil.CollectAndCompare(gauge, strings.NewReader(`
		# HELP test_gauge_enrichment h
		# TYPE test_gauge_enrichment gauge
		test_gauge_enrichment{enrichment="fetched-once",item="a"} 1
		test_gauge_enrichment{enrichment="fetched-once",item="b"} 1
	`), "test_gauge_enrichment"); err != nil {
		t.Fatalf("unexpected gauge state:\n%v", err)
	}
}

// TestRefreshScope_SetIsNeverCalledWhenFetchFails guards the ordering
// RefreshScope's pointer-sharing pattern depends on: if set ran before, or
// despite, a failed fetch, a pointer fetch was supposed to fill in (like
// flavors above) would still be at its zero value when set reads it.
func TestRefreshScope_SetIsNeverCalledWhenFetchFails(t *testing.T) {
	setCalled := false

	err := RefreshScope(
		GlobalScope{},
		"test_collector",
		func() ([]string, error) { return nil, errors.New("boom") },
		func(string) { setCalled = true },
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if setCalled {
		t.Fatal("set must not be called when fetch fails")
	}
}

func TestRefreshScope_FetchErrorIncrementsAPIErrorsCounter(t *testing.T) {
	before := testutil.ToFloat64(apiErrors.WithLabelValues("test_collector_counter"))

	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "test_gauge_counter", Help: "h"}, []string{"id"})
	_ = RefreshScope(
		GlobalScope{},
		"test_collector_counter",
		func() ([]string, error) { return nil, errors.New("boom") },
		func(string) {},
		gauge,
	)

	after := testutil.ToFloat64(apiErrors.WithLabelValues("test_collector_counter"))
	if after != before+1 {
		t.Fatalf("expected apiErrors counter to increment by 1, went from %v to %v", before, after)
	}
}
