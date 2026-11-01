package model

import (
	"crypto/subtle"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Node is a registered, credentialed participant. Tier and reputation come
// from here, never from a request payload.
type Node struct {
	ID    string
	Tier  int // 1..3; 3 (proof-exempt automatic node) only by registration
	Rep   int // 0..100
	token string
}

// Registry is the server-side node registry (token -> node).
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]Node // by ID
}

func NewRegistry() *Registry { return &Registry{nodes: map[string]Node{}} }

// Register adds or replaces a node. Tier and rep are validated.
func (r *Registry) Register(id, token string, tier, rep int) error {
	if id == "" || len(id) > 64 || token == "" || tier < 1 || tier > 3 || rep < 0 || rep > 100 {
		return fmt.Errorf("invalid node %q", id)
	}
	r.mu.Lock()
	r.nodes[id] = Node{ID: id, Tier: tier, Rep: rep, token: token}
	r.mu.Unlock()
	return nil
}

// Authenticate returns the node owning token. Every node is compared (no
// early exit on the secret) in constant time per comparison.
func (r *Registry) Authenticate(token string) (Node, bool) {
	if token == "" {
		return Node{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var found Node
	ok := false
	for _, n := range r.nodes {
		if subtle.ConstantTimeCompare([]byte(n.token), []byte(token)) == 1 {
			found, ok = n, true
		}
	}
	return found, ok
}

// DemoNodes are the scripted demo participants; their tokens are
// "dev-token-<ID>". Seeded only in mock mode.
var demoNodes = []struct {
	id        string
	tier, rep int
}{
	{"A", 2, 88}, {"B", 2, 72}, {"C", 2, 65}, {"E", 2, 66},
	{"F", 2, 63}, {"G", 2, 70}, {"H", 2, 68}, {"D", 2, 61}, {"FAO", 3, 0},
}

// LoadRegistry builds the registry from AFRIORACLE_NODES
// ("id:token:tier:rep,..."). In mock mode, when the variable is unset, the
// demo nodes are seeded with dev tokens. Real mode never seeds defaults.
func LoadRegistry() (*Registry, error) {
	r := NewRegistry()
	if spec := strings.TrimSpace(os.Getenv("AFRIORACLE_NODES")); spec != "" {
		for _, item := range strings.Split(spec, ",") {
			p := strings.Split(strings.TrimSpace(item), ":")
			if len(p) != 4 {
				return nil, fmt.Errorf("AFRIORACLE_NODES: want id:token:tier:rep, got %q", item)
			}
			tier, e1 := strconv.Atoi(p[2])
			rep, e2 := strconv.Atoi(p[3])
			if e1 != nil || e2 != nil {
				return nil, fmt.Errorf("AFRIORACLE_NODES: bad tier/rep in %q", item)
			}
			if err := r.Register(p[0], p[1], tier, rep); err != nil {
				return nil, err
			}
		}
		return r, nil
	}
	if mock {
		for _, d := range demoNodes {
			_ = r.Register(d.id, "dev-token-"+d.id, d.tier, d.rep)
		}
	}
	return r, nil
}
