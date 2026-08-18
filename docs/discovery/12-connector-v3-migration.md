# 12 — CloudView connector migration to Connector v3

Evidence base and migration design for moving `qualys_gcp_connector`,
`qualys_aws_connector` and `qualys_azure_connector` off the deprecated
CloudView v1 API (`/cloudview-api/rest/v1/...`) onto the Connector v3 APIs
under `/qps/rest/3.0/.../am/...`.

**Status: implemented** (qps/connector*.go, provider/connector_common.go and
the three resource/data-source pairs), against this document's evidence
only — no live tenant was available to this project. The open items below
are exactly what a tenant-holding contributor can confirm or refute;
corrections are welcome as issues or PRs with the observed request/response
pairs (redact credentials and ids).

Evidence grade: **Confirmed** (official docs.qualys.com pages, fetched
2026-08-16, field names quoted verbatim from request/response samples)
unless marked otherwise.

## Sources

- Index: <https://docs.qualys.com/en/conn/api/aws_3/connector_v3_apis.htm>
- AWS: `.../aws_3/connectors_aws_3.htm` and its 8 operation sub-pages
  (note: the index's `Update_AWS_Connector_3.0.htm` link 404s; the live
  page is lowercase `update_aws_connector_3.0.htm`)
- Azure: `.../azure_3/connectors_azure_3.htm` and its 6 operation sub-pages
- GCP: `.../gcp_3/connectors_gcp_3.htm` and its 7 operation sub-pages
- Deprecation notice:
  <https://notifications.qualys.com/api/2026/03/04/qualys-enterprise-trurisk-platform-totalcloud-2-22-0-and-connector-2-15-0-api-notification>

## Endpoints (Confirmed, all three clouds)

Object nouns: `awsassetdataconnector`, `azureassetdataconnector`,
`gcpassetdataconnector`. All under `/qps/rest/3.0/<verb>/am/<noun>`,
HTTP Basic auth, permission "Managers with full scope", XML default with
JSON via `Accept`/`Content-Type: application/json`.

| Verb | Path shape | Method | Notes |
|---|---|---|---|
| create | `/create/am/<noun>` | POST | |
| update | `/update/am/<noun>/<id>` | POST | id in path; response echoes **id only** — follow with get to refresh |
| get | `/get/am/<noun>/<id>` | GET | GCP page's curl shows `-X POST` here; AWS page says GET. Treat as GET, tolerate POST. ⚠️ |
| search | `/search/am/<noun>` | POST | `filters > Criteria {field, operator, value}`; only `EQUALS` documented; `preferences` paging tag; `hasMoreRecords` in response |
| delete | `/delete/am/<noun>/<id>` | POST/DELETE | single by id-in-path (no body); bulk via filters body. AWS page says POST; Azure page labels DELETE with a verb-less curl ⚠️ |
| run | `/run/am/<noun>[/<id>]` | POST (AWS/Azure) | async trigger, no body; GCP page's curl is verb-less ⚠️ |
| errors | `/search/am/assetdataconnectorerrors` | POST | shared noun across connector types; filter on `id` |

AWS extras: `GET /get/am/awsbaseaccount/<id>` + `POST /search/am/awsbaseaccount`
(response fields `globalAccountId`, `govAccountId`, `chinaAccountId`,
`customerGlobalAccount`, `customerGovAccount`, `customerChinaAccount`;
wrapper element name unconfirmed), and
`POST /download/am/awscloudformationtemplate` (request object
`AwsCloudformationTemplate` with `awsCloudType`, `externalId`, `capability`;
response is the raw CloudFormation JSON, not a ServiceResponse).

## Envelope and serialization quirks (Confirmed)

- Same `ServiceRequest`/`ServiceResponse` envelope the `qps` package
  already implements (`{"ServiceRequest": {"data": {...}}}`); JSON
  `ServiceResponse.data` is an **array** of `{"<Object>": {...}}`.
- Request collections wrap in `set` (endpoints use `add`); response
  collections wrap in `list`. DTOs need both shapes.
- Doc JSON response samples render booleans as **strings**
  (`"disabled": "false"`); requests take real booleans. Decoding must be
  tolerant (a `flexBool`).
- Object element casing is exact: `AwsAssetDataConnector`,
  `AzureAssetDataConnector`, `GcpAssetDataConnector`,
  `ConnectorAppInfoQList`, `ConnectorAppInfo`, `TagSimple`,
  `ActivationModule`, `AwsEndpointSimple`, `ConnectorScanConfiguration`.
- `connectorState` values: PENDING, SUCCESS, ERROR, QUEUED, RUNNING,
  PROCESSING, FINISHED_SUCCESS, FINISHED_ERRORS, DISABLED, INCOMPLETE.
- Update responses return only `id`; Read must re-get.
- `NOT EQUALS` is documented as unsupported/dangerous for update and
  delete filter forms; the provider only uses id-in-path forms anyway.

## Per-cloud object fields (Confirmed)

Common create/update fields: `name`, `description`,
`defaultTags` (`set > TagSimple > id`), `activation`
(`set > ActivationModule`: VM, CERTVIEW, CLOUDVIEW, SCA, CSA),
`disabled`, `runFrequency` (minutes), `isRemediationEnabled`,
`connectorAppInfos` (`set > ConnectorAppInfoQList > set >
ConnectorAppInfo {name: AI|CI|CSA, identifier, tagId}`), CPS block
(`isCPSEnabled`, `connectorScanSetting.isCustomScanConfigEnabled`,
`connectorScanConfig`: `daysOfWeek`, `optionProfileId`, `recurrence`
DAILY|WEEKLY, `scanPrefix`, `startDate` mm/dd/yyyy, `startTime` HH:MM,
`timezone`).

Common read-only fields: `id` (numeric), `connectorState`, `type`,
`lastSync`, `nextSync`, `lastError`, `isDeleted`, **`cloudviewUuid`**
(the CloudView v1 UUID — the v1→v3 identity bridge).

**AWS**: `arn`, `externalId`, `allRegions` (false ⇒ supply
`endpoints > add > AwsEndpointSimple > regionCode`),
`isGovCloudConfigured`; read-only `awsAccountId`, `qualysAwsAccountId`,
`isChinaConfigured`, `isInstantAssessmentEnabled`. All create fields
documented optional.

**Azure**: credentials nested under `authRecord`
(`AzureAuthRecordSimple`): `applicationId`, `directoryId`,
`subscriptionId`, `authenticationKey`. Create response echoes an
**empty** `authRecord` — the key is write-only in practice; do not use
reads for key drift detection. Read-only extras: `subscriptionName`,
`isGovCloudConfigured`. Search filter names use lowercase
`authrecord.applicationId` etc., unlike the body's `authRecord`.

**GCP**: create documents `name`, `type` (= GCP), `authRecord`,
`runFrequency`, `connectorAppInfos` as **required** (stricter than
AWS/Azure). `authRecord` (`GCPAuthRecordSimple`) is the Google
service-account key decomposed field-for-field; the JSON request sample
uses the key file's own snake_case names (`project_id`,
`private_key_id`, `private_key`, `client_email`, `client_id`,
`auth_uri`, `token_uri`, `auth_provider_x509_cert_url`,
`client_x509_cert_url`, `type`), while the XML sample uses `projectId`
⚠️ (XML/JSON naming differs; we use JSON, so the key file can be
embedded nearly verbatim). Reads echo only `authRecord.projectId` —
key material is never returned. Update accepts `authRecord` with only
`projectId` when the key is unchanged.

## Migration design

1. **Client**: new `qps/connector.go` (+ per-cloud DTO files) reusing
   `Client.call`/`SearchAll` — the envelope, retry, rate-limit,
   non-idempotent-create protections and https enforcement come free.
   Add a string-tolerant bool type. Delete `cloudview/` once resources
   are cut over (removes the last raw-resty client).
2. **Resources**: keep the existing names `qualys_{gcp,aws,azure}_connector`
   with a bumped `SchemaVersion`. Config-side field mapping:
   - GCP: keep `gcp_credentials_json` (the v3 JSON authRecord *is* the
     key file) — config-compatible.
   - AWS: `arn`, `external_id` map 1:1.
   - Azure: `application_id`, `directory_id`, `subscription_id`,
     `authentication_key` map into `authRecord` 1:1.
   - Add common optionals: `run_frequency`, `disabled`, `activation`,
     `default_tag_ids`, `is_remediation_enabled`. Defer the CPS scan
     block and `connectorAppInfos` to a follow-up.
3. **State/ID migration**: v1 IDs are UUIDs, v3 IDs are numeric, and v3
   responses expose `cloudviewUuid`. On Read, an ID that parses as a
   UUID triggers a one-time search of v3 connectors matching
   `cloudviewUuid` client-side (no documented server-side filter for
   it), then `d.SetId(<numeric id>)`. Fallback documented in README:
   `terraform state rm` + `terraform import <numeric-id>`.
4. **Secrets**: keep the existing write-only pattern — never overwrite
   the configured `authentication_key`/`gcp_credentials_json` from
   reads; v3 confirms the API never returns them.
5. **Update**: v3 update returns id-only, so Update must chain into
   Read (the resources already do).
6. **Data sources**: switch get-by-UUID to `get/<numeric-id>`, and add
   name-based lookup via search `Criteria {field: name, operator: EQUALS}`.

## Open items

- Verb ambiguities flagged ⚠️ above (GCP get/run, Azure delete): docs
  are internally inconsistent; validate against a live tenant before
  hard-coding anything other than the AWS-documented verbs.
- `preferences` pagination child element names are not documented on
  these pages (Qualys-wide convention `startFromOffset`/`limitResults`
  is Corroborated elsewhere in this repo's qps client).
- `awsbaseaccount` response wrapper element name unconfirmed.
- Tenant validation pass (doc 08 checklist) for: string-boolean
  responses, `cloudviewUuid` presence on search results for all three
  clouds, and whether v1 UUID-era connectors appear in v3 listings at
  all on a given subscription.
