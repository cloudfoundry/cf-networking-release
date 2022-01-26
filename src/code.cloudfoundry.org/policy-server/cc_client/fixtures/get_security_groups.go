package fixtures

const NoSecurityGroups = `{
  "pagination": {
    "total_results": 0,
    "total_pages": 1,
    "first": {
      "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
    },
    "last": {
      "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
    }
  },
  "resources": []
}`

const OneSecurityGroup = `{
  "pagination": {
    "total_results": 1,
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
      "guid": "b85a788e-671f-4549-814d-e34cdb2f539a",
      "created_at": "2020-02-20T17:42:08Z",
      "updated_at": "2020-02-20T17:42:08Z",
      "name": "my-group0",
      "globally_enabled": {
        "running": true,
        "staging": false
      },
      "rules": [
        {
          "protocol": "tcp",
          "destination": "10.10.10.0/24",
          "ports": "443,80,8080"
        },
        {
          "protocol": "icmp",
          "destination": "10.10.10.0/24",
          "type": 8,
          "code": 0,
          "description": "Allow ping requests to private services"
        }
      ],
      "relationships": {
        "staging_spaces": {
          "data": [
            { "guid": "space-guid-1" },
            { "guid": "space-guid-2" }
          ]
        },
        "running_spaces": {
          "data": [
            { "guid": "space-guid-3" },
            { "guid": "space-guid-4" }
          ]
        }
      },
      "links": {
        "self": {
          "href": "https://api.example.org/v3/security_groups/b85a788e-671f-4549-814d-e34cdb2f539a"
        }
      }
    }
  ]
}`

const TwoSecurityGroups = `{
  "pagination": {
    "total_results": 2,
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
      "guid": "b85a788e-671f-4549-814d-e34cdb2f539a",
      "created_at": "2020-02-20T17:42:08Z",
      "updated_at": "2020-02-20T17:42:08Z",
      "name": "my-group0",
      "globally_enabled": {
        "running": true,
        "staging": false
      },
      "rules": [
        {
          "protocol": "tcp",
          "destination": "10.10.10.0/24",
          "ports": "443,80,8080"
        },
        {
          "protocol": "icmp",
          "destination": "10.10.10.0/24",
          "type": 8,
          "code": 0,
          "description": "Allow ping requests to private services"
        }
      ],
      "relationships": {
        "staging_spaces": {
          "data": [
            { "guid": "space-guid-1" },
            { "guid": "space-guid-2" }
          ]
        },
        "running_spaces": {
          "data": []
        }
      },
      "links": {
        "self": {
          "href": "https://api.example.org/v3/security_groups/b85a788e-671f-4549-814d-e34cdb2f539a"
        }
      }
    },
    {
      "guid": "second-guid",
      "created_at": "2020-02-20T17:42:08Z",
      "updated_at": "2020-02-20T17:42:08Z",
      "name": "my-group2",
      "globally_enabled": {
        "running": false,
        "staging": true
      },
      "rules": [
        {
          "protocol": "tcp",
          "destination": "10.10.10.0/24",
          "ports": "53"
        }
      ],
      "relationships": {
        "staging_spaces": {
          "data": [
            { "guid": "space-guid-1" }
          ]
        },
        "running_spaces": {
          "data": [
			{ "guid": "space-guid-1" }
		  ]
        }
      },
      "links": {
        "self": {
          "href": "https://api.example.org/v3/security_groups/second-guid"
        }
      }
    }
  ]
}`

const SecurityGroupsMultiplePages = `{
  "pagination": {
    "total_results": 3,
    "total_pages": 3,
    "first": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=1&per_page=1"
    },
	"next": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=2&per_page=1"
	},
    "last": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=3&per_page=1"
    }
  },
  "resources": [
   {
      "guid": "b85a788e-671f-4549-814d-e34cdb2f539a",
      "created_at": "2020-02-20T17:42:08Z",
      "updated_at": "2020-02-20T17:42:08Z",
      "name": "my-group0",
      "globally_enabled": {
        "running": true,
        "staging": false
      },
      "rules": [
        {
          "protocol": "tcp",
          "destination": "10.10.10.0/24",
          "ports": "443,80,8080"
        },
        {
          "protocol": "icmp",
          "destination": "10.10.10.0/24",
          "type": 8,
          "code": 0,
          "description": "Allow ping requests to private services"
        }
      ],
      "relationships": {
        "staging_spaces": {
          "data": [
            { "guid": "space-guid-1" },
            { "guid": "space-guid-2" }
          ]
        },
        "running_spaces": {
          "data": []
        }
      },
      "links": {
        "self": {
          "href": "https://api.example.org/v3/security_groups/b85a788e-671f-4549-814d-e34cdb2f539a"
        }
      }
    }
  ]
}`

const SecurityGroupsMultiplePagesPg2 = `{
  "pagination": {
    "total_results": 3,
    "total_pages": 3,
    "first": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=1&per_page=1"
    },
	"next": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=3&per_page=1"
	},
    "last": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=3&per_page=1"
    }
  },
  "resources": [
    {
      "guid": "second-guid",
      "created_at": "2020-02-20T17:42:08Z",
      "updated_at": "2020-02-20T17:42:08Z",
      "name": "my-group2",
      "globally_enabled": {
        "running": false,
        "staging": true
      },
      "rules": [
        {
          "protocol": "tcp",
          "destination": "10.10.10.0/24",
          "ports": "53"
        }
      ],
      "relationships": {
        "staging_spaces": {
          "data": [
            { "guid": "space-guid-1" }
          ]
        },
        "running_spaces": {
          "data": [
			{ "guid": "space-guid-1" }
		  ]
        }
      },
      "links": {
        "self": {
          "href": "https://api.example.org/v3/security_groups/second-guid"
        }
      }
	}
  ]
}`

const SecurityGroupsMultiplePagesPg3 = `{
  "pagination": {
    "total_results": 3,
    "total_pages": 3,
    "first": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=1&per_page=1"
    },
    "last": {
      "href": "https://api.[your-domain.com]/v3/security_groups?page=3&per_page=1"
    }
  },
  "resources": [
    {
      "guid": "third-guid",
      "created_at": "2020-02-20T17:42:08Z",
      "updated_at": "2020-02-20T17:42:08Z",
      "name": "my-group3",
      "globally_enabled": {
        "running": true,
        "staging": true
      },
      "rules": [
        {
          "protocol": "tcp",
          "destination": "10.10.10.0/24",
          "ports": "123"
        }
      ],
      "relationships": {
        "staging_spaces": {
          "data": [
            { "guid": "space-guid-1" },
			{ "guid": "space-guid-2" }
          ]
        },
        "running_spaces": {
          "data": [
            { "guid": "space-guid-1" },
			{ "guid": "space-guid-2" }
		  ]
        }
      },
      "links": {
        "self": {
          "href": "https://api.example.org/v3/security_groups/third-guid"
        }
      }
	}
  ]
}`
