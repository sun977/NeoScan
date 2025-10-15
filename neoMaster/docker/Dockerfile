# Dockerfile
# demo
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o neoMaster cmd/master

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/neoMaster .
COPY --from=builder /app/configs ./configs

# 创建日志目录
RUN mkdir -p /var/log/neoscan

EXPOSE 8123

CMD ["./neoMaster"]