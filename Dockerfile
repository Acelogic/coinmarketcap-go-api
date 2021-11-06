# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

EXPOSE 8080

# Copy the local package files to the container's workspace.
COPY /src/ /go/src/

WORKDIR /go/src/

RUN go build main.go 

ENTRYPOINT /go/src/main





