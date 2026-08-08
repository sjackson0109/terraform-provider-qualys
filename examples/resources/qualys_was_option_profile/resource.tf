resource "qualys_was_option_profile" "standard" {
  name        = "Standard WAS Scan"
  performance = "MEDIUM"

  max_crawl_requests              = 2000
  timeout_error_threshold         = 10
  unexpected_error_threshold      = 10
  detect_credit_card_numbers      = true
  detect_social_security_numbers  = true
}
