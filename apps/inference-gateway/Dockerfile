FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /inference-gateway ./cmd/...

FROM gcr.io/distroless/static-debian12
COPY --from=builder /inference-gateway /inference-gateway
EXPOSE 8090
ENTRYPOINT ["/inference-gateway"]
