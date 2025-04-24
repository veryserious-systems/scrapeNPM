package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"scrapeNPM/internal/config"
	"scrapeNPM/internal/db"
	"scrapeNPM/internal/discovery"
	"scrapeNPM/internal/processor"
)

func main() {
	log.Println("Starting NPM Registry Scraper")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCh := make(chan struct{})

	var wg sync.WaitGroup

	onlyOnce := &sync.Once{}
	setupSignalHandler(cancel, shutdownCh, onlyOnce)

	cfg := config.Load()

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Successfully connected to database")

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	migrationsDir := filepath.Join(wd, "migrations")
	err = database.RunMigrations(migrationsDir)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")

	jobQueueRepo := discovery.NewJobQueueRepository(database.Pool)
	npmClient := discovery.NewClient()

	scraperCfg := discovery.DefaultConfig()
	packageScraper := discovery.NewScraper(scraperCfg, npmClient, jobQueueRepo)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting package discovery scraper...")
		if err := packageScraper.Run(ctx); err != nil {
			log.Printf("Scraper error: %v", err)
		}
	}()

	processorRepo := processor.NewRepository(database.Pool)
	numWorkers := 10

	log.Printf("Starting %d package processor workers...", numWorkers)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		worker := processor.NewWorker(i, processorRepo, npmClient, shutdownCh)
		go func(w *processor.Worker) {
			defer wg.Done()
			w.Start(ctx)
		}(worker)
	}

	<-ctx.Done()
	log.Println("Shutting down...")

	onlyOnce.Do(func() {
		close(shutdownCh)
	})

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		log.Println("All workers completed gracefully")
	case <-time.After(5 * time.Second):
		log.Println("Shutdown timed out after 5 seconds, some workers may not have completed")
	}

	log.Println("Shutdown complete")
}

func setupSignalHandler(cancel context.CancelFunc, shutdownCh chan struct{}, onlyOnce *sync.Once) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal, gracefully shutting down...")

		cancel()

		onlyOnce.Do(func() {
			close(shutdownCh)
		})
	}()
}
