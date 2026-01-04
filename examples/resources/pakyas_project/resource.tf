# Create a project to organize checks
resource "pakyas_project" "prod" {
  name        = "Production"
  description = "Production cron jobs and scheduled tasks"
}

# Reference the project ID in other resources
output "project_id" {
  value = pakyas_project.prod.id
}

# Import existing projects:
# terraform import pakyas_project.prod <project-uuid>
