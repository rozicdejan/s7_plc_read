# Use an official Go runtime with version 1.22
FROM golang:1.22-alpine

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files and download the dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code into the container
COPY . .

# Build the Go app
RUN go build -o my-go-app main.go

# Expose port 9999 to the outside world
EXPOSE 9999

# Command to run the executable
CMD ["./my-go-app"]
