FROM golang:1.21.11-bullseye AS builder
ARG BUILD_VERSION='1.2.3'
ENV LISTEN_PORT=9101
WORKDIR /opt/inventor/
COPY go.mod ./
COPY go.sum .
COPY src ./src
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.buildVersion=${BUILD_VERSION}" -a -installsuffix cgo -o inventor src/main.go

FROM debian:bullseye
WORKDIR /opt/inventor/
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    curl \
  && rm -rf /var/cache/apt/archives/*
COPY --from=builder /opt/inventor/inventor .

EXPOSE ${LISTEN_PORT}
CMD ["/opt/inventor/inventor"]
