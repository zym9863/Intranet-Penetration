FROM golang:1.24-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/chuan-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/chuan-client ./cmd/client

FROM alpine:3.20 AS runtime-base
RUN addgroup -S chuan && adduser -S chuan -G chuan
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

FROM runtime-base AS server
COPY --from=builder /out/chuan-server /usr/local/bin/chuan-server
RUN mkdir -p /app/data && chown -R chuan:chuan /app/data
USER chuan
EXPOSE 7000 80 443
ENTRYPOINT ["chuan-server"]
CMD ["-c", "/app/configs/server.docker.yaml"]

FROM runtime-base AS client
COPY --from=builder /out/chuan-client /usr/local/bin/chuan-client
USER chuan
ENTRYPOINT ["chuan-client"]
CMD ["-c", "/app/configs/client.docker.yaml"]
