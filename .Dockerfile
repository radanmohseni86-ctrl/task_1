# ==========================================
# Stage 1: Build the Go binary
# ==========================================
FROM golang:1.26-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files first (for better caching)
COPY go.mod go.sum ./
# COPY go.sum ./ (Uncomment if your project has dependencies)

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app.
# CGO_ENABLED=0 ensures a statically linked binary (vital for Alpine/Scratch).
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# ==========================================
# Stage 2: Create the minimal runtime image
# ==========================================
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080


RUN adduser -D appuser

USER appuser
# Command to run the executable
CMD ["./main"]