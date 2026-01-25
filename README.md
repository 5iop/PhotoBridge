# PhotoBridge

A photo delivery system for photographers to share photos with clients.

## Features

- **Project Management** - Organize photos by projects
- **RAW Support** - Upload and manage RAW files alongside JPG/PNG
- **Smart Matching** - Auto-link RAW and normal photos by filename
- **Share Links** - Create multiple share links per project with short URLs
- **Access Control** - Hide specific photos from individual share links
- **Download Options** - Clients can choose to download normal, RAW, or all files
- **Batch Download** - One-click ZIP download for entire photo sets

## Tech Stack

- **Backend**: Go + Gin + GORM + SQLite
- **Frontend**: Vue 3 + Vite + TailwindCSS + Pinia
- **Deployment**: Docker

## Quick Start

### Development

1. **Start backend**
```bash
cd backend
go run main.go
```

2. **Start frontend**
```bash
cd frontend
npm install
npm run dev
```

3. **Access**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8060

### Docker

```bash
docker-compose up -d
```

Access at http://localhost

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_USERNAME` | admin | Admin login username |
| `ADMIN_PASSWORD` | admin123 | Admin login password |
| `API_KEY` | photobridge-api-key | API key for programmatic uploads |
| `JWT_SECRET` | photobridge-jwt-secret | JWT signing secret |
| `PORT` | 8060 (dev) / 80 (docker) | Server port |
| `UPLOAD_DIR` | ./uploads | Photo storage directory |
| `DATABASE_PATH` | ./data/photobridge.db | SQLite database path |

## API Upload

Upload photos via API:

```bash
curl -X POST "http://localhost:8060/api/upload/ProjectName" \
  -H "X-API-Key: your-api-key" \
  -F "files=@photo1.jpg" \
  -F "files=@photo1.arw"
```

## Project Structure

```
PhotoBridge/
├── backend/
│   ├── main.go
│   ├── config/         # Configuration
│   ├── database/       # Database init
│   ├── handlers/       # API handlers
│   ├── middleware/     # Auth middleware
│   ├── models/         # Data models
│   └── utils/          # Utilities
├── frontend/
│   └── src/
│       ├── api/        # API client
│       ├── router/     # Vue Router
│       ├── stores/     # Pinia stores
│       └── views/      # Page components
├── uploads/            # Photo storage
├── data/               # SQLite database
├── Dockerfile
└── docker-compose.yml
```

## License

MIT
