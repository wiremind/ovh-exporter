package network

import (
	"testing"
)

// TestProjectIDsFromEnv pins down the parsing rules the OVH_CLOUD_PROJECT_*
// env vars rely on: comma-separated, surrounding whitespace trimmed, empty
// entries dropped (a trailing comma or an unset var must not produce a
// bogus "" project ID that would then 404 against the OVH API).
func TestProjectIDsFromEnv(t *testing.T) {
	cases := map[string]struct {
		value string
		want  []string
	}{
		"unset env var yields no project IDs":     {value: "", want: nil},
		"single project ID":                       {value: "abc123", want: []string{"abc123"}},
		"multiple project IDs":                    {value: "abc123,def456", want: []string{"abc123", "def456"}},
		"whitespace around IDs is trimmed":        {value: " abc123 , def456 ", want: []string{"abc123", "def456"}},
		"empty entries from stray commas dropped": {value: "abc123,,def456,", want: []string{"abc123", "def456"}},
		"blank value yields no project IDs":       {value: "   ", want: nil},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(EnvOVHCloudProjectInventoryProjectIDs, tc.value)

			got := projectIDsFromEnv(EnvOVHCloudProjectInventoryProjectIDs)

			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
