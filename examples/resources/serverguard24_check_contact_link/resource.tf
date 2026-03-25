resource "serverguard24_check_contact_link" "http_ops" {
  check_id   = serverguard24_check.http.id
  contact_id = serverguard24_contact.ops.id
}
