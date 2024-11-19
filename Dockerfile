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

COPY config /opt/kube

ENV KUBECONFIG="/opt/kube/config"

CMD ["/bin/controller"]