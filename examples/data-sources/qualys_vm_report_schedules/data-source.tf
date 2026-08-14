data "qualys_vm_report_schedules" "active" {
  active = true
}

output "active_schedule_titles" {
  value = [for s in data.qualys_vm_report_schedules.active.report_schedules : s.title]
}
