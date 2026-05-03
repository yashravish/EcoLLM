variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for all resources"
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone for GPU node pools"
  type        = string
  default     = "us-central1-a"
}

variable "environment" {
  description = "Deployment environment: staging or production"
  type        = string
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Environment must be staging or production."
  }
}

variable "cluster_name" {
  description = "GKE cluster name"
  type        = string
  default     = "ecollm-cluster"
}

variable "api_node_count" {
  description = "Number of CPU nodes for API services"
  type        = number
  default     = 2
}

variable "gpu_node_count" {
  description = "Number of GPU nodes for model inference"
  type        = number
  default     = 2
}

variable "gpu_machine_type" {
  description = "GPU node machine type"
  type        = string
  default     = "a2-highgpu-1g"
}

variable "db_tier" {
  description = "Cloud SQL instance tier"
  type        = string
  default     = "db-custom-4-15360"
}

variable "redis_memory_gb" {
  description = "Memorystore Redis capacity in GB"
  type        = number
  default     = 1
}
