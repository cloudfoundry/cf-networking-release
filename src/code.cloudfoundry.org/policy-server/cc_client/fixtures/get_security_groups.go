package fixtures

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

// const AppsV3LiveAppGUIDs = `{
//   "pagination": {
//     "total_results": 2,
//     "total_pages": 1,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-1-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     },
//     {
//       "guid": "live-app-2-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const SpacesV3AllGUIDs = `{
//   "pagination": {
//     "total_results": 2,
//     "total_pages": 2,
//     "first": {
//       "href": "/first_page"
//     },
//     "next": {
//       "href": "/next_page"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-1-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     },
//     {
//       "guid": "live-app-2-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3LiveApp1GUID = `{
//   "pagination": {
//     "total_results": 1,
//     "total_pages": 1,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-1-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3LiveApp2GUID = `{
//   "pagination": {
//     "total_results": 1,
//     "total_pages": 1,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-2-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3LiveApp3GUID = `{
//   "pagination": {
//     "total_results": 1,
//     "total_pages": 1,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-3-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3OneSpace = `{
//   "pagination": {
//     "total_results": 2,
//     "total_pages": 1,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-1-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     },
//     {
//       "guid": "live-app-2-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3TwoSpaces = `{
//   "pagination": {
//     "total_results": 2,
//     "total_pages": 1,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=10"
//     }
//   },
//   "resources": [
//     {
//       "guid": "live-app-1-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     },
//     {
//       "guid": "live-app-2-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-2-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3MultiplePages = `{
//   "pagination": {
//     "total_results": 3,
//     "total_pages": 3,
//     "first": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=1"
//     },
//     "last": {
//       "href": "https://api.[your-domain.com]/v3/apps?page=3&per_page=1"
//     },
// 		"next": {
// 			"href": "https://api.[your-domain.com]/v3/apps?page=2&per_page=1"
// 		}
//   },
//   "resources": [
//     {
//       "guid": "live-app-1-guid",
// 			"links": {
// 				"space": {
// 					"href": "https://api.example.org/v2/spaces/space-1-guid"
// 				}
// 			}
//     }
//   ]
// }`

// const AppsV3MultiplePagesPg2 = `{
// 	"pagination": {
// 		"total_results": 3,
// 		"total_pages": 3,
// 		"first": {
// 			"href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=1"
// 		},
// 		"last": {
// 			"href": "https://api.[your-domain.com]/v3/apps?page=3&per_page=1"
// 		},
// 		"next": {
// 			"href": "https://api.[your-domain.com]/v3/apps?page=3&per_page=1"
// 		}
// 	},
// 	"resources": [
// 	{
// 		"guid": "live-app-2-guid",
// 		"links": {
// 			"space": {
// 				"href": "https://api.example.org/v2/spaces/space-1-guid"
// 			}
// 		}
// 	}
// 	]
// }`

// const AppsV3MultiplePagesPg3 = `{
// 	"pagination": {
// 		"total_results": 3,
// 		"total_pages": 3,
// 		"first": {
// 			"href": "https://api.[your-domain.com]/v3/apps?page=1&per_page=1"
// 		},
// 		"last": {
// 			"href": "https://api.[your-domain.com]/v3/apps?page=3&per_page=1"
// 		},
// 		"next": null
// 	},
// 	"resources": [
// 	{
// 		"guid": "live-app-3-guid",
// 		"links": {
// 			"space": {
// 				"href": "https://api.example.org/v2/spaces/space-2-guid"
// 			}
// 		}
// 	}
// 	]
// }`
