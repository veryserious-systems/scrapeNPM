package discovery

import (
	"context"
	"fmt"
	"log"
	"time"

	"scrapeNPM/internal/models"
)

type Config struct {
	BatchSize    int
	RequestDelay time.Duration
	MaxRetries   int
}

func DefaultConfig() Config {
	return Config{
		BatchSize:    1000,
		RequestDelay: time.Second * 2,
		MaxRetries:   3,
	}
}

type Scraper struct {
	config         Config
	npmClient      *Client
	jobQueue       *JobQueueRepository
	lastSequence   string
	totalProcessed int64
}

func NewScraper(config Config, npmClient *Client, jobQueue *JobQueueRepository) *Scraper {
	return &Scraper{
		config:    config,
		npmClient: npmClient,
		jobQueue:  jobQueue,
	}
}

func (s *Scraper) Run(ctx context.Context) error {
	log.Printf("Starting NPM registry scraper")

	lastSeq, processed, err := s.jobQueue.GetScrapeProgress(ctx, "npm_changes")
	if err != nil {
		return fmt.Errorf("failed to get scrape progress: %w", err)
	}

	s.lastSequence = lastSeq
	s.totalProcessed = processed

	log.Printf("Resuming from sequence ID: %s (processed %d packages so far)", s.lastSequence, s.totalProcessed)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Scraper stopping due to context cancellation")
			return nil
		default:
			if err := s.processBatch(ctx); err != nil {
				log.Printf("Error processing batch: %v", err)
				time.Sleep(s.config.RequestDelay * 3)
				continue
			}

			time.Sleep(s.config.RequestDelay)
		}
	}
}

func (s *Scraper) processBatch(ctx context.Context) error {
	log.Printf("Fetching changes since %s", s.lastSequence)

	changes, err := s.npmClient.GetChanges(ctx, s.lastSequence, s.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch changes: %w", err)
	}

	results, ok := changes["results"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format: results not found or not an array")
	}

	if len(results) == 0 {
		log.Printf("No new changes found, waiting longer before next check")
		time.Sleep(time.Second * 30)
		return nil
	}

	processed := 0
	for _, result := range results {
		change, ok := result.(map[string]interface{})
		if !ok {
			continue
		}

		if deleted, ok := change["deleted"].(bool); ok && deleted {
			continue
		}

		id, ok := change["id"].(string)
		if !ok || len(id) == 0 {
			continue
		}

		if id[0] == '_' {
			continue
		}

		job := models.Job{
			Type:        "fetch_package",
			Status:      "pending",
			Priority:    5,
			MaxAttempts: 3,
			Payload: map[string]interface{}{
				"package_name": id,
				"created_at":   time.Now(),
			},
		}

		_, err := s.jobQueue.EnqueueJob(ctx, job)
		if err != nil {
			log.Printf("Failed to enqueue job for package %s: %v", id, err)
			continue
		}

		processed++
		s.totalProcessed++
	}

	log.Printf("Processed batch: queued %d new packages", processed)

	newLastSeq := ""
	if lastSeq, ok := changes["last_seq"].(string); ok {
		newLastSeq = lastSeq
	} else if lastSeqNum, ok := changes["last_seq"].(float64); ok {
		newLastSeq = fmt.Sprintf("%d", int(lastSeqNum))
	} else if lastSeqObj, ok := changes["last_seq"].(map[string]interface{}); ok {
		if seq, ok := lastSeqObj["seq"]; ok {
			if seqStr, ok := seq.(string); ok {
				newLastSeq = seqStr
			} else if seqNum, ok := seq.(float64); ok {
				newLastSeq = fmt.Sprintf("%d", int(seqNum))
			}
		}
	}

	if newLastSeq != "" && newLastSeq != s.lastSequence {
		s.lastSequence = newLastSeq

		err := s.jobQueue.UpdateScrapeProgress(ctx, "npm_changes", s.lastSequence, s.totalProcessed)
		if err != nil {
			log.Printf("Warning: failed to update scrape progress: %v", err)
		}
	}

	return nil
}
