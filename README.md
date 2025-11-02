# Cronos – CRON Manager UI

Cronos is a web-based CRON job manager built with Go and React.
It provides a simple UI to create, update, and manage scheduled jobs.

By default, jobs are stored in-memory and synced to a local SQLite file.
Optionally, Chronos can persist this SQLite file to Azure Blob Storage for durability across restarts and deployments.

## Features

- In-memory CRON scheduler
- Local SQLite persistence
- Azure Blob Storage integration for remote backup
- Simple React UI for job management
- Docker support for easy deployment

## 🛠️ Setup

You can run Chronos in two ways — using Docker (recommended for production) or locally for development.

### 🐳 Option 1: Run with Docker
If you prefer a one-command setup without installing Go or Node:
```bash
docker-compose up --build
```
The application will be available at:
http://localhost/

Docker automatically builds both Go backend and the React frontend and runs them together.

### 💻 Option 2: Local Development Setup
If you want to run and modify the backend and frontend independently:
1. Backend (Go)
  Ensure Go (1.23 or newer) is installed, then run:
```bash
go mod tidy
go run main.go
```
By default, the backend runs on port 8080.

2. Frontend (React)
Navigate to the frontend directory and start the dev server with npm or Bun.
```bash
npm install
npm run dev
```
The frontend runs on port 80 and proxies API requests to the Go backend on port 8080.

## ☁️ Azure Blob Storage Integration (Optional)

Chronos supports syncing the SQLite database to Azure Blob Storage for persistence.
When the following environment variables are set, the app will:
- 📥 Download the SQLite file from Azure Blob on startup
- 📤 Upload it back after updates or periodically

## 🔧 Environment Variables
| Variable                  | Description                    | Example             |
| ------------------------- | ------------------------------ | ------------------- |
| `AZURE_STORAGE_ACCOUNT`   | Azure Storage Account name     | `mychronosstorage`  |
| `AZURE_STORAGE_KEY`       | Storage account access key     | `<your-access-key>` |
| `AZURE_STORAGE_CONTAINER` | Blob container name            | `chronos-data`      |
| `AZURE_STORAGE_BLOB_NAME` | Blob path/name for SQLite file | `db/cron.db`        |

If these variables are not set, Chronos will fall back to local-only persistence.

### 🐳 Example docker-compose.yml
```yaml
version: "3.9"
services:
  chronos:
    build: .
    ports:
      - "80:80"
    environment:
      # Optional: Azure Blob Storage Configuration
      AZURE_STORAGE_ACCOUNT: mychronosstorage
      AZURE_STORAGE_KEY: <your-access-key>
      AZURE_STORAGE_CONTAINER: chronos-data
      AZURE_STORAGE_BLOB_NAME: db/cron.db
    volumes:
      - ./data:/app/data

```
If Azure variables are omitted, Chronos will simply persist to a local file at ./data/cron.db.

## Project Structure
```csharp
chronos/
├── backend/          # Go backend server
│   ├── main.go
│   ├── internal/
│   └── ...
├── frontend/         # React frontend
│   ├── src/
│   └── ...
├── data/             # Local SQLite file storage
├── docker-compose.yml
└── README.md

```

## Roadmap / TODO
- [ ] ☁️ Support additional blob providers
  - AWS S3
  - Google Cloud Storage
  - DigitalOcean Spaces
  - MinIO (self-hosted)
- [ ] 🔐 Add user authentication
- [ ] 📜 Add job execution logs and history
- [ ] 🧰 Expose REST API for integrations
