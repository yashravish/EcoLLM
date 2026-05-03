terraform {
  required_version = ">= 1.6"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
  backend "gcs" {
    bucket = "ecollm-terraform-state"
    prefix = "terraform/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# ── GKE Cluster ───────────────────────────────────────────────────────────────

resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.region

  remove_default_node_pool = true
  initial_node_count       = 1

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
}

resource "google_container_node_pool" "cpu_nodes" {
  name       = "cpu-pool"
  cluster    = google_container_cluster.primary.name
  location   = var.region
  node_count = var.api_node_count

  node_config {
    machine_type = "e2-standard-4"
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }

  autoscaling {
    min_node_count = 1
    max_node_count = 10
  }
}

resource "google_container_node_pool" "gpu_nodes" {
  name       = "gpu-pool"
  cluster    = google_container_cluster.primary.name
  location   = var.zone
  node_count = var.gpu_node_count

  node_config {
    machine_type = var.gpu_machine_type
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    guest_accelerator {
      type  = "nvidia-tesla-a10"
      count = 1
      gpu_driver_installation_config {
        gpu_driver_version = "LATEST"
      }
    }

    taint {
      key    = "nvidia.com/gpu"
      value  = "present"
      effect = "NO_SCHEDULE"
    }
  }
}

# ── Cloud SQL (PostgreSQL 16) ────────────────────────���─────────────────────────

resource "google_sql_database_instance" "postgres" {
  name             = "ecollm-postgres-${var.environment}"
  database_version = "POSTGRES_16"
  region           = var.region

  settings {
    tier              = var.db_tier
    availability_type = var.environment == "production" ? "REGIONAL" : "ZONAL"
    disk_size         = 50
    disk_autoresize   = true

    backup_configuration {
      enabled            = true
      start_time         = "03:00"
      retained_backups   = 7
    }

    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.vpc.id
    }
  }

  deletion_protection = var.environment == "production"
}

resource "google_sql_database" "ecollm" {
  name     = "ecollm"
  instance = google_sql_database_instance.postgres.name
}

# ── Memorystore (Redis 7) ──────────────────────────────────────────────────────

resource "google_redis_instance" "cache" {
  name           = "ecollm-redis-${var.environment}"
  tier           = "STANDARD_HA"
  memory_size_gb = var.redis_memory_gb
  region         = var.region
  redis_version  = "REDIS_7_0"
}

# ── VPC ──────────────────────────────��──────────────────────────────────���─────

resource "google_compute_network" "vpc" {
  name                    = "ecollm-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "ecollm-subnet"
  ip_cidr_range = "10.0.0.0/20"
  region        = var.region
  network       = google_compute_network.vpc.id
}
