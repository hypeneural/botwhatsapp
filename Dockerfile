# Multi-stage, stdlib-only Go binary; DO NOT deploy without operator authorization.
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/botwhatsapp ./cmd/botwhatsapp

FROM alpine:3.21
RUN addgroup -S bot && adduser -S -G bot bot
RUN mkdir -p /var/lib/botwhatsapp/spool && chown -R bot:bot /var/lib/botwhatsapp
ENV BOT_DATA_DIR=/var/lib/botwhatsapp/spool
USER bot
WORKDIR /app
COPY --from=builder /out/botwhatsapp /app/botwhatsapp
EXPOSE 8788
ENTRYPOINT ["/app/botwhatsapp"]
