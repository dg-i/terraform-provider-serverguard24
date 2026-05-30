data "serverguard24_host" "example" {
  name = "my-server.example.com"
}

output "host_address" {
  value = data.serverguard24_host.example.address
}
