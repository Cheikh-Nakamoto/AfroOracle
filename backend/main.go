package main

import (
	"fmt"
	"net/http"
	"os"

	"afrioracle/internal/controller"
	"afrioracle/internal/model"
)

func main() {
	store := model.NewInMemoryStore()
	c := controller.NewControllers(store)
	var nodes *controller.NodeRegistry
	var err error
	if os.Getenv("AFRIORACLE_DEMO_MODE") == "1" {
		nodes = controller.DemoNodeRegistry()
	} else {
		nodes, err = controller.NodeRegistryFromJSON(os.Getenv("AFRIORACLE_NODE_CREDENTIALS"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid node credential configuration:", err)
			os.Exit(1)
		}
	}
	c.Nodes = nodes

	mux := http.NewServeMux()
	mux.HandleFunc("/api/submit", controller.Cors(c.HandleSubmit))
	mux.HandleFunc("/api/consensus", controller.Cors(c.HandleConsensus))
	mux.HandleFunc("/api/state", controller.Cors(c.HandleState))
	mux.HandleFunc("/api/ask", controller.Cors(c.HandleAsk))
	mux.HandleFunc("/webhook/whatsapp", c.HandleWhatsapp)
	mux.HandleFunc("/", c.HandleRoot)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	fmt.Printf("AfriOracle server on :%s (registered nodes: %d)\n", port, nodes.Len())
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
