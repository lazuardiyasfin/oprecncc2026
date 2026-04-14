# Stage 1: Build
FROM golang:1.26.2-alpine3.23 AS builder

WORKDIR /app

# Copy these first to avoid redownloading dependencies
COPY go.mod go.sum .

# Download module, cached for faster build in consequent builds
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Using Go build cache to speed up compilation
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Stage 2: Compress
FROM hatamiarash7/upx:latest AS compressor

COPY --from=builder /app/main /workspace/main

# Compress binaries with UPX
RUN upx --best --lzma -o /workspace/main-compressed /workspace/main

# Stage 3: Final
FROM alpine:3.23 

WORKDIR /app

# Create non-root user for only running the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

RUN mkdir uploads

COPY --from=builder /app/index.html .
COPY --from=builder /app/template.html .

COPY --from=compressor /workspace/main-compressed ./main

# Manage permissions
RUN chown -R appuser:appgroup /app

USER appuser

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

EXPOSE 8080

ENTRYPOINT ["./main"]