# API's to enable
resource "google_project_service" "api_services" {
  project                    = var.project_id
  for_each                   = toset(var.api_service)
  service                    = each.key
  disable_dependent_services = true
}
