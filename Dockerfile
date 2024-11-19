FROM golang:latest AS builder

WORKDIR /opt

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

COPY ./client ./client

RUN env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o controller

FROM debian:bookworm-slim

COPY --from=builder /opt/controller /bin

RUN mkdir /opt/kube

# Default path, replace with actual value when running container
ENV KUBECONFIG_PATH="$HOME/.kube/config"

CMD ["/bin/controller"]