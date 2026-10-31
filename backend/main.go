package main

import (
	"fmt"
	"net/http"
	"os"

	"afrioracle/internal/controller"
	"afrioracle/internal/meta"
	"afrioracle/internal/model"
)

func main() {
	store := model.NewInMemoryStore()
	var ai model.AI = model.MockAI{}
	if key := os.Getenv("META_API_KEY"); key != "" {
		ai = model.NewMetaAI(meta.New(key))
	}
	c := controller.NewControllers(store, ai)

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
	mode := "real"
	if ai.Name() == "mock" {
		mode = "mock"
	}
	fmt.Printf("AfriOracle server on :%s ai=%s\n", port, mode)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
