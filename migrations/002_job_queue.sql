-- Create job queue table
CREATE TABLE IF NOT EXISTS job_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    priority INT DEFAULT 5,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    error_message TEXT,
    worker_id VARCHAR(100),
    next_attempt_after TIMESTAMP DEFAULT NOW()
);

-- Create indexes for the job queue
CREATE INDEX IF NOT EXISTS job_queue_status_idx ON job_queue(status);
CREATE INDEX IF NOT EXISTS job_queue_priority_idx ON job_queue(priority, created_at);
CREATE INDEX IF NOT EXISTS job_queue_next_attempt_idx ON job_queue(status, next_attempt_after) 
    WHERE status = 'pending';

-- Scrape progress tracking
CREATE TABLE IF NOT EXISTS scrape_progress (
    id VARCHAR(50) PRIMARY KEY,
    last_sequence VARCHAR(255),
    total_processed BIGINT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Initialize the scrape progress record for NPM changes
INSERT INTO scrape_progress (id, last_sequence, total_processed)
VALUES ('npm_changes', '0', 0)
ON CONFLICT (id) DO NOTHING;