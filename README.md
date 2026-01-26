# PhotoBridge

A photo delivery system for photographers to share photos with clients. Optimized for large files (20MB+ JPG, 60MB+ RAW) on low-memory servers.

## Features

- **Project Management** - Organize photos by projects with cover images
- **RAW Support** - Upload and manage RAW files (ARW, CR2, NEF, DNG, RAF, ORF, RW2) alongside JPG/PNG
- **Smart Matching** - Auto-link RAW and normal photos by filename
- **Thumbnail System** - Auto-generated thumbnails (400px list / 1600px preview) for fast browsing
- **EXIF Display** - View camera settings, lens info, and shooting parameters
- **Share Links** - Create multiple share links per project with custom aliases
- **Access Control** - Hide specific photos from individual share links
- **Download Options** - Clients can choose to download normal, RAW, or all files
- **Batch Download** - One-click ZIP download with streaming (no compression for already-compressed photos)
- **File Deduplication** - SHA-256 hash checking prevents duplicate uploads
- **Drag & Drop Upload** - FilePond-powered upload with progress tracking

## Performance Optimizations

Designed for 512MB RAM servers handling large photo files:

- **Chunked Hash Calculation** - 2MB chunks for SHA-256, avoids loading entire 60MB RAW into memory
- **Parallel Thumbnail Loading** - 6 concurrent requests for fast gallery rendering
- **Optimized Thumbnail Generation** - Box filter for small thumbs, CatmullRom for large (10-50x faster than Lanczos)
- **ZIP Streaming** - Store mode (no compression) reduces CPU and memory usage
- **Blob URL Caching** - Thumbnails cached as blob URLs to avoid re-fetching

## Tech Stack

- **Backend**: Go + Gin + GORM + SQLite
- **Frontend**: Vue 3 + Vite + TailwindCSS + Pinia
- **Upload**: FilePond
- **Testing**: Vitest (frontend) + Go testing (backend)
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
- Default login: admin / admin123

### Docker

```bash
docker-compose up -d
```

Access at http://localhost

### Docker with Traefik

For production deployment with Traefik reverse proxy and Let's Encrypt SSL:

```bash
# Set your domain
export PHOTOBRIDGE_DOMAIN=photos.yourdomain.com

# Start with Traefik compose file
docker-compose -f docker-compose.traefik.yml up -d
```

Requires an external Traefik network named `traefik`. See [docker-compose.traefik.yml](docker-compose.traefik.yml) for configuration.

## Testing

### Backend Tests
```bash
cd backend
go test ./...
```

### Frontend Tests
```bash
cd frontend
npm test
```

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

## API Endpoints

### Admin (JWT Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/admin/login` | Login |
| GET | `/api/admin/projects` | List projects |
| POST | `/api/admin/projects` | Create project |
| GET | `/api/admin/projects/:id` | Get project |
| PUT | `/api/admin/projects/:id` | Update project |
| DELETE | `/api/admin/projects/:id` | Delete project |
| POST | `/api/admin/projects/:id/photos` | Upload photos |
| GET | `/api/admin/projects/:id/photos` | List photos |
| POST | `/api/admin/projects/:id/photos/check-hashes` | Check for duplicates |
| DELETE | `/api/admin/photos/:id` | Delete photo |
| GET | `/api/admin/photos/:id/exif` | Get EXIF data |
| GET | `/api/admin/photos/:id/thumb/small` | Small thumbnail |
| GET | `/api/admin/photos/:id/thumb/large` | Large thumbnail |

### Share (Public)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/share/:token` | Get share info |
| GET | `/api/share/:token/photos` | List accessible photos |
| GET | `/api/share/:token/photo/:id` | Get photo |
| GET | `/api/share/:token/photo/:id/exif` | Get EXIF |
| GET | `/api/share/:token/photo/:id/download` | Download single |
| GET | `/api/share/:token/download` | Download all as ZIP |

### API (API Key Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/projects` | List all projects |
| POST | `/api/projects` | Create project |
| DELETE | `/api/projects/:name` | Delete project (must be empty) |
| GET | `/api/projects/:name/photos` | List photos with hash info |
| POST | `/api/upload/:project` | Upload photos |

**Examples:**

```bash
# List projects
curl "http://localhost:8060/api/projects" \
  -H "X-API-Key: your-api-key"

# Create project
curl -X POST "http://localhost:8060/api/projects" \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "Wedding 2024", "description": "婚礼摄影"}'

# Delete project
curl -X DELETE "http://localhost:8060/api/projects/Wedding%202024" \
  -H "X-API-Key: your-api-key"

# Get project photos
curl "http://localhost:8060/api/projects/Wedding%202024/photos" \
  -H "X-API-Key: your-api-key"

# Upload photos
curl -X POST "http://localhost:8060/api/upload/ProjectName" \
  -H "X-API-Key: your-api-key" \
  -F "files=@photo1.jpg" \
  -F "files=@photo1.arw"
```

**API Documentation:** Access Swagger UI at `http://localhost:8060/api/docs`

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
│   └── utils/          # Utilities (zip, hash, thumbnail)
├── frontend/
│   └── src/
│       ├── api/        # API client
│       ├── components/ # Reusable components
│       ├── router/     # Vue Router
│       ├── stores/     # Pinia stores
│       ├── views/      # Page components
│       └── __tests__/  # Vitest tests
├── uploads/            # Photo storage
├── data/               # SQLite database
├── Dockerfile
└── docker-compose.yml
```

## License

MIT
