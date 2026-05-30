# Note: The serverguard24_check data source only supports lookup by ID
# due to API limitations. Use the serverguard24_check resource for
# managing new checks.

data "serverguard24_check" "example" {
  id = "12345"
}

output "check_address" {
  value = data.serverguard24_check.example.address
}

output "check_shortname" {
  value = data.serverguard24_check.example.shortname
}