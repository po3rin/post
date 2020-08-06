FROM golang:1.14.3 as builder

WORKDIR /src

COPY go.mod /src/go.mod
COPY go.sum /src/go.sum

RUN go mod download

# Perform the build
COPY ./cmd/postctl ./cmd/postctl
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /dist/postctl ./cmd/postctl


FROM alpine/git:v2.24.3
COPY --from=builder /dist/postctl /usr/local/bin/postctl
