# scrapeNPM

A high-performance, multi-threaded NPM registry scraper and analyzer that replicates the NPM package registry into a PostgreSQL database with a focus on installation scripts.

Developed by [VerySerious Systems](https://github.com/veryserious-systems)

## 📋 Overview

scrapeNPM discovers, processes, and stores NPM packages in a PostgreSQL database, with a particular focus on extracting and analyzing installation scripts (`preinstall`, `install`, and `postinstall`). It uses a robust, multi-threaded architecture with a job queuing system to efficiently process npm packages.

### Key Features

- 🔍 Complete discovery of all NPM registry packages via the changes feed
- 📊 Extract and store scripts for security analysis and auditing
- 📥 Uses PostgreSQL as both a job queue and persistent store
- 🧵 Multi-threaded processing with configurable worker count
- 🔄 Fault-tolerant with automatic retries and job recovery
- 📈 Tracks NPM package metadata, including version history and download statistics
- 🔁 Resumable operations via sequence checkpointing

## 🛠️ Architecture

scrapeNPM is designed with a clean, modular architecture:

1. **Discovery**: Detects new and updated packages from the NPM registry
2. **Processing**: Fetches package details and extracts relevant data
3. **Storage**: Persists package information in a PostgreSQL database

The system uses a durable job queue pattern, where jobs are stored in the database and processed by worker threads. This ensures reliable processing even if the application is restarted.

## 🚀 Getting Started

### Prerequisites

- Go 1.18+
- PostgreSQL 13+
- Git

### Installation

1. Clone the repository:

```bash
git clone https://github.com/veryserious-systems/scrapeNPM.git
cd scrapeNPM
```

2. Install dependencies:

```bash
go mod tidy
```

3. Configure the database connection in db.go

4. Build the project:

```bash
# For development
go build -o scrapeNPM ./cmd/scraper

# For production
./build.sh
```

### Running

```bash
./scrapeNPM
```

The application will:

1. Connect to the database and run any pending migrations
2. Start discovering packages from the NPM registry
3. Queue jobs for package processing
4. Process packages and extract scripts
5. Store data in the database

## 🗂️ Database Schema

The database schema includes:

- `packages`: Core package metadata
- `package_scripts`: Installation scripts for packages
- `job_queue`: Processing queue for asynchronous operations
- `scrape_progress`: Tracking for incremental scraping progress

## 🔧 Configuration

Configuration is editable in db.go before building

## 📝 Usage Examples

### Find packages with suspicious install scripts

```sql
SELECT p.name, ps.script_type, ps.content 
FROM packages p
JOIN package_scripts ps ON p.id = ps.package_id
WHERE ps.content LIKE '%curl%' OR ps.content LIKE '%wget%'
ORDER BY p.downloads DESC;
```

### Get the most popular packages

```sql
SELECT name, version, downloads, popularity_score
FROM packages
ORDER BY downloads DESC
LIMIT 100;
```

## 🛡️ Security Considerations

This tool scrapes and stores installation scripts that may contain security-sensitive commands. Always run in an isolated environment and be careful when examining script content.

## 🧪 Testing

```bash
go test ./...
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- NPM Registry for providing public access to package metadata
- The Go community for exceptional libraries and tools

---

Made with ❤️ by [veryserious.systems](veryserious.systems)
