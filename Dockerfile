FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Create a minimal image for running the application
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/server .

# Copy the config file
COPY config.yml .

# Create a directory for uploads
RUN mkdir -p /app/uploads

# Set environment variables with defaults
ENV PORT=8080
ENV JWT_SECRET=your-jwt-secret-key
ENV MINIO_ENDPOINT=minio:9000
ENV MINIO_ACCESS_KEY=minioadmin
ENV MINIO_SECRET_KEY=minioadmin
ENV MINIO_BUCKET=uploads
ENV MINIO_USE_SSL=false
ENV TUS_BASE_PATH=/files/
ENV STORAGE_PATH=/app/uploads

EXPOSE 8080

# Run the application
CMD ["/app/server"] 