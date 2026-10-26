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
	fmt.Println("AfriOracle server on :" + port)
	http.ListenAndServe(":"+port, mux)
}
