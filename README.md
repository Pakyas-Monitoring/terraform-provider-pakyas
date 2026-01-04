# Terraform Provider for Pakyas

The Pakyas provider enables Terraform to manage [Pakyas](https://pakyas.com) cron job monitoring resources.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (for building from source)

## Installation

### From Terraform Registry (Coming Soon)

```hcl
terraform {
  required_providers {
    pakyas = {
      source  = "pakyas/pakyas"
      version = "~> 0.1"
    }
  }
}
```

### From Source

```bash
git clone https://github.com/pakyas/terraform-provider-pakyas.git
cd terraform-provider-pakyas
make install
```

## Authentication

The provider requires an API key to authenticate. You can obtain one from the [Pakyas dashboard](https://pakyas.com) under **Settings > API Keys**.

Set the API key via environment variable (recommended):

```bash
export PAKYAS_API_KEY="pk_live_..."
```

Or in the provider configuration (not recommended for production):

```hcl
provider "pakyas" {
  api_key = "pk_live_..."
}
```

## Usage

### Provider Configuration

```hcl
provider "pakyas" {
  # API key can be set via PAKYAS_API_KEY environment variable
  # api_key = "pk_live_..."

  # Optional: Override API URL (defaults to https://api.pakyas.com)
  # api_url = "https://api.pakyas.com"
}
```

### Create a Project

```hcl
resource "pakyas_project" "prod" {
  name        = "Production"
  description = "Production cron jobs"
}
```

### Create a Check

```hcl
resource "pakyas_check" "daily_backup" {
  project_id     = pakyas_project.prod.id
  name           = "Daily Backup"
  slug           = "daily-backup"
  period_seconds = 86400    # 24 hours
  grace_seconds  = 3600     # 1 hour
  description    = "Daily database backup"
  tags           = ["backup", "database"]
}

output "ping_url" {
  value = pakyas_check.daily_backup.ping_url
}
```

### Import Existing Resources

```bash
# Import a project
terraform import pakyas_project.prod <project-uuid>

# Import a check
terraform import pakyas_check.daily_backup <check-uuid>
```

## Resources

### pakyas_project

Manages a Pakyas project.

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Project name (1-100 characters) |
| `description` | string | No | Project description (max 500 characters) |
| `id` | string | Computed | Project UUID |
| `org_id` | string | Computed | Organization UUID |
| `created_at` | string | Computed | Creation timestamp |
| `updated_at` | string | Computed | Last update timestamp |

### pakyas_check

Manages a Pakyas health check.

#### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `project_id` | string | Yes | Parent project UUID (ForceNew) |
| `name` | string | Yes | Check name (1-100 characters) |
| `slug` | string | Yes | Unique slug within project (ForceNew) |
| `period_seconds` | int | Yes | Expected ping interval (60-2,592,000) |
| `grace_seconds` | int | No | Grace period before alerting (0-86,400, default: 0) |
| `description` | string | No | Check description (max 500 characters) |
| `tags` | set(string) | No | Tags for organizing checks |
| `paused` | bool | No | Whether check is paused (default: false) |
| `id` | string | Computed | Check UUID |
| `public_id` | string | Computed | Public ping ID |
| `ping_url` | string | Computed | Full ping URL |
| `status` | string | Computed | Current status (new, up, down, late, paused) |
| `created_at` | string | Computed | Creation timestamp |

## Development

### Building

```bash
make build
```

### Installing Locally

```bash
make install
```

### Running Tests

```bash
# Unit tests
make test

# Acceptance tests (requires PAKYAS_API_KEY)
export PAKYAS_API_KEY="pk_test_..."
make testacc
```

### Linting

```bash
make lint
```

## License

Mozilla Public License 2.0
