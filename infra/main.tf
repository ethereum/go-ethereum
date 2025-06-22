resource "google_service_account" "lime_account" {
  account_id   = "${var.project_id}-account"
  display_name = "Lime GKE Account"
}
