data "serverguard24_integrations" "all" {}

output "integrations" {
  value = data.serverguard24_integrations.all.integrations
}
