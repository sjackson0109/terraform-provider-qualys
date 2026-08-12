---
page_title: "Additional qualys_auth_record_* Resources - terraform-provider-qualys"
subcategory: ""
description: |-
  Authentication record resources for every additional Confirmed record type.
---

# Additional `qualys_auth_record_*` resources

`qualys_auth_record_windows` and `qualys_auth_record_unix` each have their own
page because Windows carries extra `domain`/`domain_type` fields. Every other
Confirmed authentication record type (this repository's discovery notes,
doc 03 §11) shares the identical schema `qualys_auth_record_unix` documents —
`title`, `username`, `password`/`vault_id`/`vault_type`, `ips`, `comments`,
plus computed `created`/`modified` — so rather than 28 near-duplicate pages,
they are documented together here. See
[`qualys_auth_record_windows`](auth_record_windows.md) for the full schema
reference, credential write-only behaviour, and import instructions, all of
which apply identically to every resource below.

| Resource | Record type slug | Technology |
|---|---|---|
| `qualys_auth_record_network_ssh` | `network_ssh` | Generic network device (SSH) |
| `qualys_auth_record_oracle` | `oracle` | Oracle Database |
| `qualys_auth_record_oracle_listener` | `oracle_listener` | Oracle Listener |
| `qualys_auth_record_snmp` | `snmp` | SNMP |
| `qualys_auth_record_mssql` | `ms_sql` | Microsoft SQL Server |
| `qualys_auth_record_mysql` | `mysql` | MySQL |
| `qualys_auth_record_postgresql` | `postgresql` | PostgreSQL |
| `qualys_auth_record_sybase` | `sybase` | Sybase |
| `qualys_auth_record_ibm_db2` | `ibm_db2` | IBM DB2 |
| `qualys_auth_record_informix` | `informixdb` | Informix |
| `qualys_auth_record_mariadb` | `mariadb` | MariaDB |
| `qualys_auth_record_mongodb` | `mongodb` | MongoDB |
| `qualys_auth_record_neo4j` | `neo4j` | Neo4j |
| `qualys_auth_record_sapiq` | `sapiq` | SAP IQ |
| `qualys_auth_record_sap_hana` | `sap_hana` | SAP HANA |
| `qualys_auth_record_vmware` | `vmware` | VMware ESX/ESXi |
| `qualys_auth_record_vcenter` | `vcenter` | VMware vCenter |
| `qualys_auth_record_http` | `http` | HTTP |
| `qualys_auth_record_apache` | `apache` | Apache HTTP Server |
| `qualys_auth_record_nginx` | `nginx` | Nginx |
| `qualys_auth_record_ms_iis` | `ms_iis` | Microsoft IIS |
| `qualys_auth_record_ibm_websphere` | `ibm_websphere` | IBM WebSphere |
| `qualys_auth_record_tomcat` | `tomcat` | Apache Tomcat |
| `qualys_auth_record_oracle_weblogic` | `oracle_weblogic` | Oracle WebLogic |
| `qualys_auth_record_jboss` | `jboss` | JBoss |
| `qualys_auth_record_kubernetes` | `kubernetes` | Kubernetes |
| `qualys_auth_record_docker` | `docker` | Docker |
| `qualys_auth_record_palo_alto` | `palo_alto_firewall` | Palo Alto Firewall |

Note: doc 03 §11 also records that Cisco IOS and Checkpoint Firewall targets
are authenticated through the `unix` endpoint rather than a dedicated one —
use `qualys_auth_record_unix` for those, not a type-specific resource.

## Example Usage

```terraform
resource "qualys_auth_record_mysql" "app_db" {
  title    = "app-db-readonly"
  username = "qualys_scanner"
  password = var.mysql_scanner_password

  ips = ["10.50.0.0/24"]
}

resource "qualys_auth_record_kubernetes" "cluster01" {
  title    = "cluster01-service-account"
  username = "qualys-scanner"

  vault_id   = data.qualys_vaults.corp.vaults[0].id
  vault_type = data.qualys_vaults.corp.vaults[0].type

  ips = ["10.60.0.0/24"]
}
```

## Import

Import any of these by ID, using its own resource type:

```shell
terraform import qualys_auth_record_mysql.app_db 234567
```
