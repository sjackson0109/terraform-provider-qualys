# Audit existing schedules, or reference one created outside Terraform.
data "qualys_vm_scan_schedules" "weekly" {
  title_contains = "Weekly"
}

output "weekly_schedule_ids" {
  value = [for s in data.qualys_vm_scan_schedules.weekly.scan_schedules : s.id]
}
