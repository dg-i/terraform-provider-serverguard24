data "serverguard24_integrations" "all" {}

resource "serverguard24_check_integration_link" "http_jira" {
  check_id       = serverguard24_check.http.id
  integration_id = data.serverguard24_integrations.all.integrations[0].id
}
