# Look up an existing web application to reference in a scan schedule
# without managing it here.
data "qualys_was_applications" "storefront" {
  name = "Storefront"
}

output "storefront_id" {
  value = one([for a in data.qualys_was_applications.storefront.web_applications : a.id])
}
