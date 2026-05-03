# EcoLLM Deployment Guide

## Local Development

### Prerequisites

- Docker 24+
- Docker Compose v2
- Go 1.22+ (for API development)
- Node.js 20+ (for web development)
- Python 3.11+ (for prompt-optimizer)

### Start all services

```bash
make setup      # Install dependencies
make migrate    # Run database migrations
make seed       # Seed model registry and grid data
docker-compose -f infra/docker/docker-compose.yml up
```

### With GPU inference (requires NVIDIA GPU + drivers)

```bash
docker-compose -f infra/docker/docker-compose.yml \
               -f infra/docker/docker-compose.gpu.yml up
```

---

## Staging Deployment (Kubernetes)

### Prerequisites

- `kubectl` configured for your cluster
- `kustomize` 5+
- Images pushed to your container registry

### Deploy

```bash
kubectl apply -k infra/k8s/overlays/staging
```

### Verify

```bash
kubectl get pods -n ecollm-staging
kubectl logs -n ecollm-staging deploy/staging-ecollm-api
```

---

## Production Deployment

### Infrastructure provisioning (Terraform)

```bash
cd infra/terraform
terraform init
terraform plan -var="project_id=my-gcp-project" -var="environment=production"
terraform apply
```

### Deploy application

```bash
# Tag and push images
make docker-build-push ENV=production

# Apply Kubernetes manifests
kubectl apply -k infra/k8s/overlays/production
```

### Run database migrations

```bash
bash scripts/migrate.sh $DATABASE_URL
```

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `JWT_SECRET` | Yes | 32-byte random secret for JWT signing |
| `GRID_API_KEY` | Yes | Electricity Maps API key |
| `PHI3_SIDECAR_URL` | No | URL of Python prompt-optimizer sidecar |
| `INFERENCE_PHI3_URL` | Yes | vLLM endpoint for Phi-3 |
| `INFERENCE_MISTRAL_URL` | Yes | vLLM endpoint for Mistral-7B |
| `INFERENCE_LLAMA13B_URL` | Yes | vLLM endpoint for Llama-13B |
| `INFERENCE_LLAMA70B_URL` | Yes | vLLM endpoint for Llama-70B |
| `GRID_REGION` | No | Default grid region (default: `US-EAST`) |
| `LOG_LEVEL` | No | `debug` \| `info` \| `warn` (default: `info`) |

---

## Health Checks

- `GET /health` — liveness: returns 200 if server is running
- `GET /ready` — readiness: pings PostgreSQL and Redis, returns 200 only if both are reachable

---

## Scaling

The HPA in `infra/k8s/base/hpa.yaml` scales:
- API: 2–10 replicas at 70% CPU
- Web: 2–5 replicas at 70% CPU

GPU nodes are manually scaled based on traffic patterns; GPU inference does not auto-scale.
