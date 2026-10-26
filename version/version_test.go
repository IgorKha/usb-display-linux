package version

import "testing"

func TestInfo(t *testing.T) {
	info := Info()

	keys := []string{"version", "git_commit", "build_time", "go_version", "os", "arch"}
	for _, key := range keys {
		value, ok := info[key]
		if !ok {
			t.Fatalf("Info missing key %q", key)
		}
		if value == "" {
			t.Fatalf("Info key %q has empty value", key)
		}
	}
}
