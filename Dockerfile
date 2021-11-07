# Start the Go app build
FROM golang:latest AS build

# Copy source
WORKDIR /go/src/
# Copy the local package files to the container's workspace.
COPY /src/ /go/src/

WORKDIR /go/src/

# Get required modules (assumes packages have been added to ./vendor)
RUN go mod download

# Build a statically-linked Go binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .

# New build phase -- create binary-only image
FROM alpine:latest

# Add support for HTTPS
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates

WORKDIR /

# Copy files from previous build container
COPY --from=build /go/src/main ./


# Check results
RUN env && pwd && find .

EXPOSE 8080

# Start the application
CMD ["./main"]




