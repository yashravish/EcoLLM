FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /carbon-service ./cmd/...

FROM gcr.io/distroless/static-debian12
COPY --from=builder /carbon-service /carbon-service
EXPOSE 8091
ENTRYPOINT ["/carbon-service"]
