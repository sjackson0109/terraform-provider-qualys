# Look up an existing web application to reference in a scan schedule
# without managing it here.
data "qualys_web_applications" "storefront" {
  name = "Storefront"
}

output "storefront_id" {
  value = one([for a in data.qualys_web_applications.storefront.web_applications : a.id])
}
