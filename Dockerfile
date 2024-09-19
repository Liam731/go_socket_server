# Build stage
FROM golang:1.22-alpine3.19 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the rest of the source code into the container
COPY . .

# Build the Go application
RUN go build -o main main.go

# Runtime stage
FROM alpine:3.19

# Set the working directory inside the container
WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=builder /app/main .

# Copy the environment variable file
COPY app.env .

# Expose port 8080 for the application
EXPOSE 8080

# Run the Go application
CMD ["/app/main"]
