data "serverguard24_distributors" "all" {}

resource "serverguard24_check_distributor_link" "http_dist" {
  check_id       = serverguard24_check.http.id
  distributor_id = data.serverguard24_distributors.all.distributors[0].id
}
