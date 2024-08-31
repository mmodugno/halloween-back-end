# Use an official Golang image as the base image
FROM golang:1.22-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download and cache the dependencies
RUN go mod download

# Copy the entire project directory to the container
COPY . .

# Build the Go app, with the binary named "backend"
RUN go build -o backend ./cmd

# Expose the port on which the backend runs
EXPOSE 8080

# Command to run the backend
CMD ["./backend"]