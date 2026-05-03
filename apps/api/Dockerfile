FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/server/...

# Test stage (used by docker-compose.test.yml)
FROM builder AS test
CMD ["go", "test", "-race", "./..."]

FROM gcr.io/distroless/static-debian12 AS production
COPY --from=builder /api /api
EXPOSE 8080 9091
ENTRYPOINT ["/api"]
