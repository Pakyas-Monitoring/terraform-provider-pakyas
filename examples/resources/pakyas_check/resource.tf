# Create a daily backup check
resource "pakyas_check" "daily_backup" {
  project_id     = pakyas_project.prod.id
  name           = "Daily Backup"
  slug           = "daily-backup"
  period_seconds = 86400  # 24 hours
  grace_seconds  = 3600   # 1 hour grace period
  description    = "Daily database backup job"
  tags           = ["backup", "database"]
}

# Create a check that runs every 5 minutes
resource "pakyas_check" "health_monitor" {
  project_id     = pakyas_project.prod.id
  name           = "Health Monitor"
  slug           = "health-monitor"
  period_seconds = 300  # 5 minutes
  grace_seconds  = 60   # 1 minute grace period
}

# A paused check (useful for maintenance)
resource "pakyas_check" "weekly_report" {
  project_id     = pakyas_project.prod.id
  name           = "Weekly Report"
  slug           = "weekly-report"
  period_seconds = 604800  # 1 week
  paused         = true    # Temporarily disabled
}

# Output the ping URL for use in cron jobs
output "backup_ping_url" {
  value       = pakyas_check.daily_backup.ping_url
  description = "Ping URL for the daily backup check"
}

# Import existing checks:
# terraform import pakyas_check.daily_backup <check-uuid>
