# ============================================================
# Build stage
# ============================================================
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -tags="netgo osusergo timetzdata" \
    -ldflags="-s -w" \
    -o /app/notification-service \
    ./cmd/server


# ============================================================
# Runtime stage
# ============================================================
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/notification-service /app/notification-service

EXPOSE 8080

ENTRYPOINT ["/app/notification-service"]