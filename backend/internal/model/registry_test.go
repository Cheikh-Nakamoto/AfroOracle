package model

import "testing"

func TestLoadRegistryFromEnv(t *testing.T) {
	t.Setenv("AFRIORACLE_NODES", "N1:s3cret:2:75,AUTO:t0k:3:0")
	r, err := LoadRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := r.Authenticate("s3cret"); !ok || n.ID != "N1" || n.Tier != 2 || n.Rep != 75 {
		t.Fatalf("N1: %+v %v", n, ok)
	}
	if _, ok := r.Authenticate("dev-token-A"); ok {
		t.Fatal("demo nodes seeded despite explicit config")
	}
	t.Setenv("AFRIORACLE_NODES", "bad")
	if _, err := LoadRegistry(); err == nil {
		t.Fatal("want error for malformed spec")
	}
}

func TestMockModeSeedsDemoNodes(t *testing.T) {
	t.Setenv("AFRIORACLE_NODES", "")
	r, _ := LoadRegistry()
	if mock {
		if n, ok := r.Authenticate("dev-token-FAO"); !ok || n.Tier != 3 {
			t.Fatalf("FAO: %+v %v", n, ok)
		}
	}
	if _, ok := r.Authenticate(""); ok {
		t.Fatal("empty token authenticated")
	}
}
