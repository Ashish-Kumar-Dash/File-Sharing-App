<h1 align="center">File Sharing App</h1>
<p align="center">
Room-based file sharing over your local network. Create a room, share the 6-character code, and everyone with the code can upload, download, and delete files. Rooms auto-expire after 24 hours.
</p>
<p align="center">
    <img src="https://img.shields.io/badge/License-MIT-yellow" alt="License: MIT" />
</p>

---

## Table of Contents
1. [How It Works](#how-it-works)
2. [Tech Stack](#tech-stack)
3. [Quick Start (Docker)](#quick-start-docker)
4. [Local Development](#local-development)
5. [API Reference](#api-reference)
6. [Project Structure](#project-structure)
7. [Contributing](#contributing)
8. [License](#license)

---

## How It Works

1. **Create a room** — the server generates a unique 6-character code (e.g. `A7K2M9`).
2. **Share the code** — anyone on the network can join by entering it.
3. **Upload & download files** — drag-and-drop on web, file picker on mobile.
4. **Auto-cleanup** — MinIO lifecycle policy deletes all room files after 24 hours.

No accounts, no sign-up, no internet required. Works entirely over your local network.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Flutter (web + mobile) |
| Backend | Go (stdlib `net/http`) |
| Storage | MinIO (S3-compatible object storage) |
| Infra | Docker Compose |

---

## Quick Start (Docker)

```bash
git clone https://github.com/OpenLake/File-Sharing-App.git
cd File-Sharing-App
docker compose up --build
```

| Service | URL |
|---------|-----|
| App | http://localhost:3000 |
| API | http://localhost:8000 |
| MinIO Console | http://localhost:9001 (user: `minioadmin` / pass: `minioadmin123`) |

---

## Local Development

### Prerequisites
- [Go](https://go.dev) >= 1.21
- [Flutter](https://flutter.dev) >= 3.10
- [MinIO](https://min.io) server running locally

### MinIO
```bash
mkdir -p ~/minio-data
minio server ~/minio-data --console-address :9001
```

### Backend
```bash
cd Go
cat > .env <<EOF
LOCAL_IP=localhost:9000
ACCESS_KEY=minioadmin
SECRET_KEY=minioadmin123
EOF
go run file-uploader.go
```

### Frontend
```bash
cd filesharing
flutter pub get
flutter run -d chrome --dart-define=API_BASE_URL=http://localhost:8000
```

### Tests
```bash
cd Go && go test -v ./...
cd filesharing && flutter test
```

---

## API Reference

All endpoints accept and return JSON unless noted.

| Method | Path | Query Params | Description |
|--------|------|-------------|-------------|
| `GET` | `/health` | — | Health check, returns `OK` |
| `POST` | `/rooms` | — | Create a room, returns `{"room": "A7K2M9"}` |
| `POST` | `/upload` | `room` | Upload a file (multipart form, field: `file`) |
| `GET` | `/download` | `room`, `filename` | Get a presigned download URL |
| `GET` | `/files` | `room` | List files in a room |
| `DELETE` | `/files` | `room`, `name` | Delete a file from a room |

**Room codes:** 6 characters, uppercase alphanumeric excluding ambiguous characters (`0`, `O`, `1`, `I`, `L`).

**File constraints:**
- Max size: 100 MB
- Blocked extensions: `.exe`, `.bat`, `.cmd`, `.sh`, `.ps1`, `.msi`
- Path traversal characters (`..`, `/`, `\`) rejected in filenames

---

## Project Structure

```
.
├── Go/
│   ├── file-uploader.go       # All backend handlers
│   ├── file-uploader_test.go   # 14 integration tests
│   ├── Dockerfile
│   └── go.mod
├── filesharing/
│   └── lib/
│       ├── main.dart           # App entry, room state, theme
│       ├── room_screen.dart    # Create/join room screen
│       ├── file_list_screen.dart # File browser, upload, download, delete
│       └── theme.dart          # Light/dark theme definitions
├── docker-compose.yml
└── README.md
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

MIT — see [LICENSE](LICENSE).
