package fixtures

const Spaces = `{
  "total_results": 2,
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
        "name": "name-2050",
        "organization_guid": "d154425c-dccc-42e6-b6b4-27d46c3b42cb",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/d154425c-dccc-42e6-b6b4-27d46c3b42cb",
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
    },
    {
      "metadata": {
        "guid": "2e100106-0b74-4062-8671-0d375f951cb5",
        "url": "/v2/spaces/2e100106-0b74-4062-8671-0d375f951cb4",
        "created_at": "2016-06-08T16:41:40Z",
        "updated_at": "2016-06-08T16:41:26Z"
      },
      "entity": {
        "name": "name-2051",
        "organization_guid": "d154425c-dccc-42e6-b6b4-27d46c3b42cb",
        "space_quota_definition_guid": null,
        "allow_ssh": true,
        "organization_url": "/v2/organizations/d154425c-dccc-42e6-b6b4-27d46c3b42cb",
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

const SpaceV3LiveSpaces = `{
   "pagination": {
      "total_results": 2,
      "total_pages": 1,
      "first": {
        "href": "https://foo.bar/v3/spaces?page=1"
      },
      "last": {
        "href": "https://foo.bar/v3/spaces?page=1"
      },
      "next": null,
      "previous": null
   },
   "resources": [
      {
         "guid": "live-space-1-guid",
         "created_at": "2018-07-24T17:49:02Z",
         "updated_at": "2018-07-24T17:49:02Z",
         "name": "space-1",
         "relationships": {
            "organization": {
               "data": {
                  "guid": "3638bc38-4e7a-45c9-8119-40af6f58b088"
               }
            }
         }
      },
      {
         "guid": "live-space-2-guid",
         "created_at": "2018-07-24T17:49:02Z",
         "updated_at": "2018-07-24T17:49:02Z",
         "name": "space-2",
         "relationships": {
            "organization": {
               "data": {
                  "guid": "3638bc38-4e7a-45c9-8119-40af6f58b088"
               }
            }
         }
      }
   ]
}`

const SpaceV3MultiplePages = `{
   "pagination": {
      "total_results": 2,
      "total_pages": 2,
      "first": {
        "href": "https://foo.bar/v3/spaces?page=1"
      },
      "last": {
        "href": "https://foo.bar/v3/spaces?page=2"
      },
      "next": {
        "href": "https://foo.bar/v3/spaces?page=2"
      },
      "previous": null
   },
   "resources": [
      {
         "guid": "live-space-1-guid",
         "created_at": "2018-07-24T17:49:02Z",
         "updated_at": "2018-07-24T17:49:02Z",
         "name": "space-1",
         "relationships": {
            "organization": {
               "data": {
                  "guid": "3638bc38-4e7a-45c9-8119-40af6f58b088"
               }
            }
         }
      },
      {
         "guid": "live-space-2-guid",
         "created_at": "2018-07-24T17:49:02Z",
         "updated_at": "2018-07-24T17:49:02Z",
         "name": "space-2",
         "relationships": {
            "organization": {
               "data": {
                  "guid": "3638bc38-4e7a-45c9-8119-40af6f58b088"
               }
            }
         }
      }
   ]
}`
