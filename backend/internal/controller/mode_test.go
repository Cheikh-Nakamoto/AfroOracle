package controller

import (
	"net/http"
	"testing"
)

func TestModeEndpointReportsActiveModes(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()

	code, out := get(t, ts.URL+"/api/mode")
	if code != http.StatusOK {
		t.Fatalf("mode status = %d, want 200: %v", code, out)
	}
	for _, key := range []string{"ai", "chain"} {
		if value, ok := out[key].(string); !ok || value == "" {
			t.Errorf("mode response %q = %v, want a non-empty string", key, out[key])
		}
	}
}
