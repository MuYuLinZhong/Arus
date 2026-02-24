FROM golang:1.22-alpine AS builder

WORKDIR /app

# 构建时代理（由 docker-compose build.args 传入，可选）
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}

RUN apk add --no-cache git

COPY go.mod go.sum ./
# 使用国内可访问的模块代理，避免直连 proxy.golang.org 超时；走代理时跳过 sum 校验
ENV GOPROXY=https://goproxy.cn,direct
RUN GONOSUMDB=* go mod download

COPY . .
ARG GOARCH=arm64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build -ldflags="-s -w" -o /lock-service ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /lock-service /lock-service
COPY migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/lock-service"]
