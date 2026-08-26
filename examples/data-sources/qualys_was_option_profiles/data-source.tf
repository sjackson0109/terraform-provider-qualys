# Reference an option profile managed outside Terraform, e.g. one shipped by
# default with the subscription.
data "qualys_was_option_profiles" "initial" {
  name_contains = "Initial"
}

output "initial_profile_id" {
  value = one([for p in data.qualys_was_option_profiles.initial.option_profiles : p.id])
}
