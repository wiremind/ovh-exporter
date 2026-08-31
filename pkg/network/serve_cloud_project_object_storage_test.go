package network

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type storageStub struct {
	regions map[string]string
	storage map[string]string
}

func newStorageTestClient(t *testing.T, stub storageStub) *ovh.Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/time", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "%d", time.Now().Unix())
	})
	mux.HandleFunc("GET /cloud/project/{projectID}/region", func(w http.ResponseWriter, r *http.Request) {
		writeStubBody(w, stub.regions[r.PathValue("projectID")])
	})
	mux.HandleFunc("GET /cloud/project/{projectID}/region/{regionName}/storage", func(w http.ResponseWriter, r *http.Request) {
		writeStubBody(w, stub.storage[r.PathValue("projectID")+"/"+r.PathValue("regionName")])
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := ovh.NewClient(server.URL, "app-key", "app-secret", "consumer-key")
	if err != nil {
		t.Fatalf("failed to create test OVH client: %v", err)
	}

	return client
}

func writeStubBody(w http.ResponseWriter, body string) {
	if body == "" {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"message":"stubbed failure"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, body)
}

// resetObjectStorageState puts the package-level state a test depends on
// back to what it looks like at the start of a refresh cycle. Flushing
// ovhAPICache is what updateMetrics does per cycle: without it a test
// would read the region list a previous test cached under the same
// project id and silently skip the regions it just stubbed.
func resetObjectStorageState(t *testing.T) {
	t.Helper()
	reset := func() {
		cloudProjectObjectStorageObjects.Reset()
		cloudProjectObjectStorageBytes.Reset()
		ovhAPICache.Flush()
	}
	reset()
	t.Cleanup(reset)
}

func assertObjectStorageGauges(t *testing.T, wantObjects, wantBytes string) {
	t.Helper()

	if err := testutil.CollectAndCompare(
		cloudProjectObjectStorageObjects,
		strings.NewReader(wantObjects),
		"ovh_exporter_cloud_project_object_storage_objects",
	); err != nil {
		t.Fatalf("unexpected objects gauge state:\n%v", err)
	}
	if err := testutil.CollectAndCompare(
		cloudProjectObjectStorageBytes,
		strings.NewReader(wantBytes),
		"ovh_exporter_cloud_project_object_storage_bytes",
	); err != nil {
		t.Fatalf("unexpected bytes gauge state:\n%v", err)
	}
}

func TestUpdateCloudProjectObjectStorage_BucketProducesBothMetrics(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	client := newStorageTestClient(t, storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[{"name":"backups","region":"EU-WEST-PAR","objectsCount":42,"objectsSize":1337}]`,
		},
	})

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 42
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 1337
	`)
}

func TestUpdateCloudProjectObjectStorage_MultipleBucketsAreIndependentSeries(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	client := newStorageTestClient(t, storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[
				{"name":"backups","region":"EU-WEST-PAR","objectsCount":42,"objectsSize":1337},
				{"name":"logs","region":"EU-WEST-PAR","objectsCount":7,"objectsSize":99}
			]`,
		},
	})

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 42
		ovh_exporter_cloud_project_object_storage_objects{bucket="logs",project_id="project-a",region="EU-WEST-PAR"} 7
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 1337
		ovh_exporter_cloud_project_object_storage_bytes{bucket="logs",project_id="project-a",region="EU-WEST-PAR"} 99
	`)
}

func TestUpdateCloudProjectObjectStorage_RegionLabelFollowsQueriedRegion(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	client := newStorageTestClient(t, storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR","RBX"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[{"name":"shared","objectsCount":1,"objectsSize":10}]`,
			"project-a/RBX":         `[{"name":"shared","region":"RBX","objectsCount":2,"objectsSize":20}]`,
		},
	})

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="shared",project_id="project-a",region="EU-WEST-PAR"} 1
		ovh_exporter_cloud_project_object_storage_objects{bucket="shared",project_id="project-a",region="RBX"} 2
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="shared",project_id="project-a",region="EU-WEST-PAR"} 10
		ovh_exporter_cloud_project_object_storage_bytes{bucket="shared",project_id="project-a",region="RBX"} 20
	`)
}

func TestUpdateCloudProjectObjectStorage_LargeSizesKeepFullPrecision(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	client := newStorageTestClient(t, storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[{"name":"archive","region":"EU-WEST-PAR","objectsCount":3000000000,"objectsSize":9895604649984}]`,
		},
	})

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="archive",project_id="project-a",region="EU-WEST-PAR"} 3e+09
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="archive",project_id="project-a",region="EU-WEST-PAR"} 9.895604649984e+12
	`)
}

func TestUpdateCloudProjectObjectStorage_FailingRegionDoesNotSuppressOthers(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	client := newStorageTestClient(t, storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR","RBX","DE"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[{"name":"backups","region":"EU-WEST-PAR","objectsCount":42,"objectsSize":1337}]`,
			"project-a/DE":          `[{"name":"logs","region":"DE","objectsCount":7,"objectsSize":99}]`,
		},
	})

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 42
		ovh_exporter_cloud_project_object_storage_objects{bucket="logs",project_id="project-a",region="DE"} 7
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 1337
		ovh_exporter_cloud_project_object_storage_bytes{bucket="logs",project_id="project-a",region="DE"} 99
	`)
}

func TestUpdateCloudProjectObjectStorage_FailingProjectDoesNotSuppressOthers(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a,project-b")

	client := newStorageTestClient(t, storageStub{
		regions: map[string]string{"project-b": `["EU-WEST-PAR"]`},
		storage: map[string]string{
			"project-b/EU-WEST-PAR": `[{"name":"backups","region":"EU-WEST-PAR","objectsCount":42,"objectsSize":1337}]`,
		},
	})

	before := testutil.ToFloat64(apiErrors.WithLabelValues(CollectorCloudProjectObjectStorage))

	updateCloudProjectObjectStorage(client)

	if after := testutil.ToFloat64(apiErrors.WithLabelValues(CollectorCloudProjectObjectStorage)); after != before+1 {
		t.Fatalf("expected the failed region listing to increment apiErrors once, went from %v to %v", before, after)
	}

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="backups",project_id="project-b",region="EU-WEST-PAR"} 42
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="backups",project_id="project-b",region="EU-WEST-PAR"} 1337
	`)
}

func TestUpdateCloudProjectObjectStorage_DeletedBucketSeriesAreDropped(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	stub := storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[
				{"name":"backups","region":"EU-WEST-PAR","objectsCount":42,"objectsSize":1337},
				{"name":"logs","region":"EU-WEST-PAR","objectsCount":7,"objectsSize":99}
			]`,
		},
	}
	client := newStorageTestClient(t, stub)

	updateCloudProjectObjectStorage(client)

	stub.storage["project-a/EU-WEST-PAR"] = `[{"name":"backups","region":"EU-WEST-PAR","objectsCount":43,"objectsSize":2000}]`

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 43
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 2000
	`)
}

func TestUpdateCloudProjectObjectStorage_FailedRefreshKeepsPreviousValues(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "project-a")

	stub := storageStub{
		regions: map[string]string{"project-a": `["EU-WEST-PAR"]`},
		storage: map[string]string{
			"project-a/EU-WEST-PAR": `[{"name":"backups","region":"EU-WEST-PAR","objectsCount":42,"objectsSize":1337}]`,
		},
	}
	client := newStorageTestClient(t, stub)

	updateCloudProjectObjectStorage(client)

	delete(stub.storage, "project-a/EU-WEST-PAR")

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, `
		# HELP ovh_exporter_cloud_project_object_storage_objects Number of objects stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_objects gauge
		ovh_exporter_cloud_project_object_storage_objects{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 42
	`, `
		# HELP ovh_exporter_cloud_project_object_storage_bytes Number of bytes stored in an OVHcloud Object Storage bucket.
		# TYPE ovh_exporter_cloud_project_object_storage_bytes gauge
		ovh_exporter_cloud_project_object_storage_bytes{bucket="backups",project_id="project-a",region="EU-WEST-PAR"} 1337
	`)
}

func TestUpdateCloudProjectObjectStorage_NoProjectsConfigured(t *testing.T) {
	resetObjectStorageState(t)
	t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, "")

	client := newStorageTestClient(t, storageStub{})

	updateCloudProjectObjectStorage(client)

	assertObjectStorageGauges(t, "", "")
}
