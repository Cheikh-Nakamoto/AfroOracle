package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"afrioracle/internal/controller"
	"afrioracle/internal/model"
)

func main() {
	store := model.NewInMemoryStore()
	c := controller.NewControllers(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           controller.NewRouter(c, controller.DefaultMaxBody),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ai, chain := model.Mode()
	fmt.Printf("AfriOracle server on :%s (ai=%s, chain=%s)\n", port, ai, chain)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
