resource "serverguard24_downtime" "maintenance" {
  hostname    = serverguard24_host.web.name
  start_time  = "2026-04-01T00:00:00Z"
  end_time    = "2026-04-01T02:00:00Z"
  description = "Scheduled maintenance window"

  services = [tonumber(serverguard24_check.http.id)]
}
