package fixtures

const AppsV3 = `{
  "pagination": {
    "total_results": 5,
    "total_pages": 1,
    "first": {
      "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
    },
    "last": {
      "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
    }
  },
  "resources": [
    {
      "guid": "live-app-1-guid",
			"links": {
				"space": {
					"href": "https://api.example.org/v2/spaces/space-1-guid"
				}
			}
    },
    {
      "guid": "live-app-2-guid",
			"links": {
				"space": {
					"href": "https://api.example.org/v2/spaces/space-1-guid"
				}
			}
    },
    {
      "guid": "live-app-3-guid",
			"links": {
				"space": {
					"href": "https://api.example.org/v2/spaces/space-2-guid"
				}
			}
    },
    {
      "guid": "live-app-4-guid",
			"links": {
				"space": {
					"href": "https://api.example.org/v2/spaces/space-2-guid"
				}
			}
    },
    {
      "guid": "live-app-5-guid",
			"links": {
				"space": {
					"href": "https://api.example.org/v2/spaces/space-3-guid"
				}
			}
    }
  ]
}`
