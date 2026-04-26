data "serverguard24_integrations" "all" {}

output "integrations" {
  value = data.serverguard24_integrations.all.integrations
}

output "by_name" {
  value = data.serverguard24_integrations.all.by_name
}
