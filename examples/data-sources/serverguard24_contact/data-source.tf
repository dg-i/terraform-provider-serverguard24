data "serverguard24_contact" "example" {
  name = "Ops Team"
}

output "contact_email" {
  value = data.serverguard24_contact.example.email
}
