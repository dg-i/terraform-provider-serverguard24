data "serverguard24_nodes" "all" {}

output "nodes" {
  value = data.serverguard24_nodes.all.nodes
}
