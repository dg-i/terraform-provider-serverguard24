data "serverguard24_groups" "all" {}

output "groups" {
  value = data.serverguard24_groups.all.groups
}
