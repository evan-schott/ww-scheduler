# Use the official Golang base image
FROM golang:1.19

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire project directory to the working directory
COPY . .

# Build the project
RUN go build -o ww-load-balancer ./cmd/lb/main.go

# Expose the port the application will run on
EXPOSE 8080

# Run the compiled binary
CMD ["./ww-load-balancer"]
