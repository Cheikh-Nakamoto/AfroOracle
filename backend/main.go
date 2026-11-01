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
	var store model.Store = model.NewInMemoryStore()
	storeKind := "memory"
	if url := os.Getenv("DATABASE_URL"); url != "" {
		pg, err := model.NewPGStore(url)
		if err != nil {
			log.Fatalf("postgres store: %v", err)
		}
		defer pg.Close()
		store, storeKind = pg, "postgres"
	}
	reg, err := model.LoadRegistry()
	if err != nil {
		log.Fatalf("node registry: %v", err)
	}
	c := controller.NewControllers(store, reg)

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
	fmt.Printf("AfriOracle server on :%s (ai=%s, chain=%s, store=%s)\n", port, ai, chain, storeKind)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
