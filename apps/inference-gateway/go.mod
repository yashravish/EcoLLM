module github.com/ecollm/inference-gateway

go 1.22

require (
	github.com/gofiber/fiber/v2 v2.52.4
	github.com/rs/zerolog v1.32.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0
	github.com/prometheus/client_golang v1.19.0
)
