variable "config_file" {
  description = "Path to the canonical JSON generated from the named Excel data tables."
  type        = string
  default     = "qualys.auto.json"
}

variable "qualys_concurrency" {
  description = "Optional maximum number of concurrent Qualys API calls. Leave null to use the provider default."
  type        = number
  default     = null

  validation {
    # try() is required, not stylistic: without it this throws "argument
    # must not be null" on the variable's own null default, since HCL
    # evaluates both sides of || here rather than short-circuiting away
    # from the null comparison.
    condition     = var.qualys_concurrency == null || try(var.qualys_concurrency > 0, false)
    error_message = "qualys_concurrency must be null or greater than zero."
  }
}

variable "qualys_max_retries" {
  description = "Optional retry count for non-destructive Qualys API calls affected by rate or concurrency limits. Leave null to use the provider default."
  type        = number
  default     = null

  validation {
    condition     = var.qualys_max_retries == null || try(var.qualys_max_retries >= 0, false)
    error_message = "qualys_max_retries must be null or zero or greater."
  }
}

variable "resolve_references_by_name" {
  description = <<-EOT
    Look up pre-existing Qualys objects (networks, asset groups, scanner
    appliances, tags, vaults, WAS web applications/report templates/
    authentication records/DNS overrides, static search lists, VM and WAS
    option profiles, admin users, roles) by name/username/title via the
    Qualys API, to resolve an external_*_name reference field to the id
    the provider actually needs. This is what makes "no numeric Qualys ids
    anywhere in the workbook" work for objects a row merely references
    rather than manages.

    Every lookup is a plain reference resolution, not an import: a name
    that doesn't match an existing object simply fails at plan time with
    an ordinary "key does not exist" error, not a changed plan.

    Requires a live Qualys API call on every plan/apply that has at least
    one enabled row in a table using an external_*_name field, so it is
    disabled (set to false) for offline testing against fixture JSON with
    no real Qualys credentials. Leave it at the default (true) for real
    runs.
  EOT
  type        = bool
  default     = true
}
