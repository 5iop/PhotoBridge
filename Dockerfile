# ============================================
# Stage 1: Build Frontend
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files first for better caching
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci --prefer-offline --no-audit

# Copy source files
COPY frontend/ ./

# Build frontend
RUN npm run build

# ============================================
# Stage 2: Build Backend
# ============================================
FROM golang:1.21-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app/backend

# Copy go mod files first for better caching
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy source files
COPY backend/ ./

# Build binary with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o photobridge \
    .

# ============================================
# Stage 3: Production Image
# ============================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 photobridge

WORKDIR /app

# Copy binary from backend builder
COPY --from=backend-builder /app/backend/photobridge .

# Copy frontend build from frontend builder
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create directories for data and uploads
RUN mkdir -p /app/data /app/uploads && \
    chown -R photobridge:photobridge /app

# Switch to non-root user
USER photobridge

# Environment variables
ENV PORT=8080 \
    UPLOAD_DIR=/app/uploads \
    DATABASE_PATH=/app/data/photobridge.db \
    GIN_MODE=release

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run
CMD ["./photobridge"]
