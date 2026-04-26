# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code from the context (project root)
COPY . .

# Build the application
RUN go build -o main ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main .
# Copy migrations for potential in-container use
COPY --from=builder /app/migrations ./migrations

# Expose the port
EXPOSE 8080

# Command to run
CMD ["./main"]
