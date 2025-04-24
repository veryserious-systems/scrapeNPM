package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"scrapeNPM/internal/discovery"
	"scrapeNPM/internal/models"
)

type Worker struct {
	id           int
	repo         *Repository
	npmClient    *discovery.Client
	extractor    *Extractor
	shutdownCh   <-chan struct{}
	workerID     string
	pollingDelay time.Duration
}

func NewWorker(id int, repo *Repository, npmClient *discovery.Client, shutdownCh <-chan struct{}) *Worker {
	return &Worker{
		id:           id,
		repo:         repo,
		npmClient:    npmClient,
		extractor:    NewExtractor(),
		shutdownCh:   shutdownCh,
		workerID:     fmt.Sprintf("worker-%d", id),
		pollingDelay: 1 * time.Second,
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Printf("[Worker %d] Starting package processor worker", w.id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Worker %d] Shutting down: context cancelled", w.id)
			return
		case <-w.shutdownCh:
			log.Printf("[Worker %d] Shutting down: shutdown signal received", w.id)
			return
		default:
			job, err := w.repo.ClaimJob(ctx, w.workerID)
			if err != nil {
				log.Printf("[Worker %d] Error claiming job: %v", w.id, err)
				time.Sleep(w.pollingDelay)
				continue
			}

			if job == nil {
				time.Sleep(w.pollingDelay)
				continue
			}

			log.Printf("[Worker %d] Processing job %s of type %s", w.id, job.ID, job.Type)
			err = w.processJob(ctx, job)

			if err != nil {
				log.Printf("[Worker %d] Failed to process job %s: %v", w.id, job.ID, err)
				if err := w.repo.FailJob(ctx, job.ID, err.Error()); err != nil {
					log.Printf("[Worker %d] Error marking job as failed: %v", w.id, err)
				}
			} else {
				log.Printf("[Worker %d] Completed job %s", w.id, job.ID)
				if err := w.repo.CompleteJob(ctx, job.ID); err != nil {
					log.Printf("[Worker %d] Error marking job as completed: %v", w.id, err)
				}
			}
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *models.Job) error {
	switch job.Type {
	case "fetch_package":
		return w.processFetchPackageJob(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func (w *Worker) processFetchPackageJob(ctx context.Context, job *models.Job) error {
	pkgName, ok := job.Payload["package_name"].(string)
	if !ok || pkgName == "" {
		return fmt.Errorf("invalid package name in job payload")
	}

	log.Printf("[Worker %d] Fetching package: %s", w.id, pkgName)
	rawPackage, err := w.npmClient.GetPackage(ctx, pkgName)
	if err != nil {
		return fmt.Errorf("failed to fetch package data: %w", err)
	}

	pkg, err := w.extractor.ExtractPackageData(pkgName, rawPackage)
	if err != nil {
		return fmt.Errorf("failed to extract package data: %w", err)
	}

	log.Printf("[Worker %d] Fetching download count for: %s", w.id, pkgName)
	downloads, docsErr := w.fetchDownloadCount(ctx, pkgName)
	if docsErr != nil {
		log.Printf("[Worker %d] Warning: failed to fetch download count for %s: %v", w.id, pkgName, docsErr)
		downloads = 0
	}
	pkg.Downloads = downloads
	pkg.PopularityScore = w.extractor.CalculatePopularityScore(downloads)

	log.Printf("[Worker %d] Storing package: %s", w.id, pkgName)
	packageID, err := w.repo.StorePackage(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to store package: %w", err)
	}

	log.Printf("[Worker %d] Extracting scripts for package: %s", w.id, pkgName)
	scripts, err := w.extractor.ExtractScripts(rawPackage, packageID, pkg.Version)
	if err != nil {
		log.Printf("[Worker %d] Warning: failed to extract scripts for %s: %v", w.id, pkgName, err)
	} else {
		for _, script := range scripts {
			log.Printf("[Worker %d] Storing %s script for package: %s", w.id, script.ScriptType, pkgName)
			if err := w.repo.StoreScript(ctx, script); err != nil {
				log.Printf("[Worker %d] Warning: failed to store %s script for %s: %v",
					w.id, script.ScriptType, pkgName, err)
			}
		}
	}

	return nil
}

func (w *Worker) fetchDownloadCount(ctx context.Context, pkgName string) (int64, error) {
	url := fmt.Sprintf("https://api.npmjs.org/downloads/point/last-month/%s", pkgName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch download stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var downloadInfo struct {
		Downloads int64 `json:"downloads"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&downloadInfo); err != nil {
		return 0, fmt.Errorf("failed to decode download stats: %w", err)
	}

	return downloadInfo.Downloads, nil
}
