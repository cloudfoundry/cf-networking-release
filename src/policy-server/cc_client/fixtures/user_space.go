package fixtures

const SubjectSpace = `{
  "total_results": 1,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [
    {
      "metadata": {
        "guid": "2e100106-0b74-4062-8671-0d375f951cb4",
        "url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "some-space-name",
        "organization_guid": "some-org-guid",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/some-org-guid",
        "developers_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/developers",
        "managers_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/managers",
        "auditors_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/auditors",
        "apps_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/apps",
        "routes_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/routes",
        "domains_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/domains",
        "service_instances_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/service_instances",
        "app_events_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/app_events",
        "events_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/events",
        "security_groups_url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4/security_groups"
      }
    }
  ]
}`

const SubjectSpaceEmpty = `{
  "total_results": 0,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": []
}`

const SubjectSpaces = `{
  "total_results": 2,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": [
    {
      "metadata": {
        "guid": "space-1-guid",
        "url": "/v2/spaces/space-1-guid",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "space-1-name",
        "organization_guid": "org-1-guid",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/org-1-guid",
        "developers_url": "/v2/spaces/space-1-guid/developers",
        "managers_url": "/v2/spaces/space-1-guid/managers",
        "auditors_url": "/v2/spaces/space-1-guid/auditors",
        "apps_url": "/v2/spaces/space-1-guid/apps",
        "routes_url": "/v2/spaces/space-1-guid/routes",
        "domains_url": "/v2/spaces/space-1-guid/domains",
        "service_instances_url": "/v2/spaces/space-1-guid/service_instances",
        "app_events_url": "/v2/spaces/space-1-guid/app_events",
        "events_url": "/v2/spaces/space-1-guid/events",
        "security_groups_url": "/v2/spaces/space-1-guid/security_groups"
      }
    },
    {
      "metadata": {
        "guid": "space-2-guid",
        "url": "/v2/spaces/space-2-guid",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "space-2-name",
        "organization_guid": "org-2-guid",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/org-2-guid",
        "developers_url": "/v2/spaces/space-2-guid/developers",
        "managers_url": "/v2/spaces/space-2-guid/managers",
        "auditors_url": "/v2/spaces/space-2-guid/auditors",
        "apps_url": "/v2/spaces/space-2-guid/apps",
        "routes_url": "/v2/spaces/space-2-guid/routes",
        "domains_url": "/v2/spaces/space-2-guid/domains",
        "service_instances_url": "/v2/spaces/space-2-guid/service_instances",
        "app_events_url": "/v2/spaces/space-2-guid/app_events",
        "events_url": "/v2/spaces/space-2-guid/events",
        "security_groups_url": "/v2/spaces/space-2-guid/security_groups"
      }
    }
  ]
}`

const SubjectSpacesPage1 = `{
  "total_results": 3,
  "total_pages": 3,
  "prev_url": null,
  "next_url": "/v2/users/some-subject-id/spaces?order-direction=asc&page=2&results-per-page=1",
  "resources": [
    {
      "metadata": {
        "guid": "space-1-guid",
        "url": "/v2/spaces/space-1-guid",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "space-1-name",
        "organization_guid": "org-1-guid",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/org-1-guid",
        "developers_url": "/v2/spaces/space-1-guid/developers",
        "managers_url": "/v2/spaces/space-1-guid/managers",
        "auditors_url": "/v2/spaces/space-1-guid/auditors",
        "apps_url": "/v2/spaces/space-1-guid/apps",
        "routes_url": "/v2/spaces/space-1-guid/routes",
        "domains_url": "/v2/spaces/space-1-guid/domains",
        "service_instances_url": "/v2/spaces/space-1-guid/service_instances",
        "app_events_url": "/v2/spaces/space-1-guid/app_events",
        "events_url": "/v2/spaces/space-1-guid/events",
        "security_groups_url": "/v2/spaces/space-1-guid/security_groups"
      }
    }
  ]
}`
const SubjectSpacesPage2 = `{
  "total_results": 3,
  "total_pages": 3,
  "prev_url": "/v2/users/some-subject-id/spaces?order-direction=asc&page=1&results-per-page=1",
  "next_url": "/v2/users/some-subject-id/spaces?order-direction=asc&page=3&results-per-page=1",
  "resources": [
    {
      "metadata": {
        "guid": "space-2-guid",
        "url": "/v2/spaces/space-2-guid",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "space-2-name",
        "organization_guid": "org-2-guid",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/org-2-guid",
        "developers_url": "/v2/spaces/space-2-guid/developers",
        "managers_url": "/v2/spaces/space-2-guid/managers",
        "auditors_url": "/v2/spaces/space-2-guid/auditors",
        "apps_url": "/v2/spaces/space-2-guid/apps",
        "routes_url": "/v2/spaces/space-2-guid/routes",
        "domains_url": "/v2/spaces/space-2-guid/domains",
        "service_instances_url": "/v2/spaces/space-2-guid/service_instances",
        "app_events_url": "/v2/spaces/space-2-guid/app_events",
        "events_url": "/v2/spaces/space-2-guid/events",
        "security_groups_url": "/v2/spaces/space-2-guid/security_groups"
      }
    }
  ]
}`
const SubjectSpacesPage3 = `{
  "total_results": 3,
  "total_pages": 3,
  "prev_url": "/v2/users/some-subject-id/spaces?order-direction=asc&page=2&results-per-page=1",
  "next_url": null,
  "resources": [
    {
      "metadata": {
        "guid": "space-3-guid",
        "url": "/v2/spaces/space-3-guid",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "space-3-name",
        "organization_guid": "org-3-guid",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/org-3-guid",
        "developers_url": "/v2/spaces/space-3-guid/developers",
        "managers_url": "/v2/spaces/space-3-guid/managers",
        "auditors_url": "/v2/spaces/space-3-guid/auditors",
        "apps_url": "/v2/spaces/space-3-guid/apps",
        "routes_url": "/v2/spaces/space-3-guid/routes",
        "domains_url": "/v2/spaces/space-3-guid/domains",
        "service_instances_url": "/v2/spaces/space-3-guid/service_instances",
        "app_events_url": "/v2/spaces/space-3-guid/app_events",
        "events_url": "/v2/spaces/space-3-guid/events",
        "security_groups_url": "/v2/spaces/space-3-guid/security_groups"
      }
    }
  ]
}`
