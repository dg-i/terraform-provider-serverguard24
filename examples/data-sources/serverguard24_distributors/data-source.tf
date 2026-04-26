data "serverguard24_distributors" "all" {}

output "distributors" {
  value = data.serverguard24_distributors.all.distributors
}

output "by_name" {
  value = data.serverguard24_distributors.all.by_name
}
