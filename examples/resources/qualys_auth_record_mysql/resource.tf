resource "qualys_auth_record_mysql" "app_db" {
  title    = "app-db-readonly"
  username = "qualys_scanner"
  password = var.mysql_scanner_password

  ips = ["10.50.0.0/24"]
}
