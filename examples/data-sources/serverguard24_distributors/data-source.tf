data "serverguard24_distributors" "all" {}

output "distributors" {
  value = data.serverguard24_distributors.all.distributors
}
