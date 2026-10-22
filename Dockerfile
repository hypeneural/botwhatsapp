# Production build uses a SECURITY-SUPPORTED compiler, even though go.mod
# deliberately retains Go 1.23 language compatibility for local Antigravity.
# 2026-10-22: Go 1.27.3 fixes crypto/x509; Alpine 3.24.2 is patched.
FROM golang:1.27.3-alpine3.24 AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/botwhatsapp ./cmd/botwhatsapp

FROM alpine:3.24.2
# CA certificates are needed by https://evydencia.com. tzdata provides
# America/Sao_Paulo, without falling back to a guessed fixed UTC offset.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S bot && adduser -S -G bot bot \
    && mkdir -p /var/lib/botwhatsapp/spool \
    && chown -R bot:bot /var/lib/botwhatsapp
ENV BOT_DATA_DIR=/var/lib/botwhatsapp/spool \
    TZ=America/Sao_Paulo
WORKDIR /app
COPY --from=builder /out/botwhatsapp /app/botwhatsapp
USER bot
EXPOSE 8788
ENTRYPOINT ["/app/botwhatsapp"]
