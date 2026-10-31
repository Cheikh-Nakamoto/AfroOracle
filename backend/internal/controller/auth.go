package controller

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// NodeCredential is trusted server-side metadata for one registered agent.
// APIKey is never returned by an HTTP endpoint.
type NodeCredential struct {
	APIKey   string `json:"api_key"`
	Tier     int    `json:"tier"`
	RepScore int    `json:"rep_score"`
}

// NodeRegistry authenticates bearer keys and supplies authoritative node
// metadata. It is loaded once during server startup.
type NodeRegistry struct {
	byID map[string]NodeCredential
}

func EmptyNodeRegistry() *NodeRegistry {
	return &NodeRegistry{byID: make(map[string]NodeCredential)}
}

func NewNodeRegistry(nodes map[string]NodeCredential) (*NodeRegistry, error) {
	reg := EmptyNodeRegistry()
	seenKeys := make(map[string]bool, len(nodes))
	for id, credential := range nodes {
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("node id cannot be empty")
		}
		if len(credential.APIKey) < 32 {
			return nil, fmt.Errorf("API key for node %q must be at least 32 characters", id)
		}
		if credential.Tier < 1 || credential.Tier > 3 {
			return nil, fmt.Errorf("tier for node %q must be 1, 2, or 3", id)
		}
		if credential.RepScore < 0 || credential.RepScore > 100 {
			return nil, fmt.Errorf("rep_score for node %q must be between 0 and 100", id)
		}
		if seenKeys[credential.APIKey] {
			return nil, fmt.Errorf("API keys must be unique")
		}
		seenKeys[credential.APIKey] = true
		reg.byID[id] = credential
	}
	return reg, nil
}

func NodeRegistryFromJSON(raw string) (*NodeRegistry, error) {
	if strings.TrimSpace(raw) == "" {
		return EmptyNodeRegistry(), nil
	}
	var nodes map[string]NodeCredential
	if err := json.Unmarshal([]byte(raw), &nodes); err != nil {
		return nil, fmt.Errorf("decode AFRIORACLE_NODE_CREDENTIALS: %w", err)
	}
	return NewNodeRegistry(nodes)
}

// DemoNodeRegistry contains public, demo-only keys. It is enabled only by an
// explicit AFRIORACLE_DEMO_MODE=1 setting and must never be used in deployment.
func DemoNodeRegistry() *NodeRegistry {
	nodes := map[string]NodeCredential{}
	for id, values := range map[string][2]int{
		"A": {2, 88}, "B": {2, 72}, "C": {2, 65}, "D": {2, 61},
		"E": {2, 66}, "F": {2, 63}, "G": {2, 70}, "H": {2, 68},
		"FAO": {3, 0},
	} {
		nodes[id] = NodeCredential{APIKey: demoKey(id), Tier: values[0], RepScore: values[1]}
	}
	registry, _ := NewNodeRegistry(nodes) // constants above are validated at build time by tests
	return registry
}

func demoKey(nodeID string) string { return "afrioracle-demo-key-v1-" + nodeID + "-d7f3a0c948b16e2d" }

func (r *NodeRegistry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.byID)
}

// Authenticate accepts only a strict Bearer header and scans every registered
// key using constant-time comparisons.
func (r *NodeRegistry) Authenticate(header string) (string, NodeCredential, bool) {
	if r == nil || !strings.HasPrefix(header, "Bearer ") {
		return "", NodeCredential{}, false
	}
	provided := sha256.Sum256([]byte(strings.TrimPrefix(header, "Bearer ")))
	matchedID := ""
	var matched NodeCredential
	valid := 0
	for id, credential := range r.byID {
		stored := sha256.Sum256([]byte(credential.APIKey))
		if subtle.ConstantTimeCompare(provided[:], stored[:]) == 1 {
			matchedID, matched = id, credential
			valid = 1
		}
	}
	return matchedID, matched, valid == 1
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error":%q}`, message)
}

func (c *Controllers) authenticate(r *http.Request) (string, NodeCredential, bool) {
	return c.Nodes.Authenticate(r.Header.Get("Authorization"))
}
