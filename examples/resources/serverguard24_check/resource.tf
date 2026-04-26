resource "serverguard24_check" "http" {
  host_id   = serverguard24_host.web.id
  type      = "http"
  shortname = "HTTP"

  params {
    path    = "/"
    port    = 443
    timeout = 10
  }
}
