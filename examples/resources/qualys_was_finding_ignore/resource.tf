data "qualys_was_findings" "storefront_active" {
  web_app_id = qualys_web_application.storefront.id
  status     = "ACTIVE"
}

resource "qualys_was_finding_ignore" "known_false_positive" {
  finding_id = one([
    for f in data.qualys_was_findings.storefront_active.findings :
    f.id if f.qid == "150012"
  ])
  comment = "False positive, verified by security team on 2026-08-16."
}
