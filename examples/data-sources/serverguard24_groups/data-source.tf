data "serverguard24_groups" "all" {}

output "groups" {
  value = data.serverguard24_groups.all.groups
}

output "by_name" {
  value = data.serverguard24_groups.all.by_name
}
