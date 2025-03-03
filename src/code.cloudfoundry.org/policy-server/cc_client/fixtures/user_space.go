package fixtures

const SubjectSpace = `{
  "guid": "885735b5-aea4-4cf5-8e44-961af0e41920",
  "created_at": "2017-02-01T01:33:58Z",
  "updated_at": "2017-02-01T01:33:58Z",
  "name": "some-space-name,
  "relationships": {
    "organization": {
      "data": {
        "guid": "e00705b9-7b42-4561-ae97-2520399d2133"
      }
    },
    "quota": {
      "data": null
    }
  },
  "links": {
    "self": {
      "href": "https://api.example.org/v3/spaces/885735b5-aea4-4cf5-8e44-961af0e41920"
    },
    "features": {
      "href": "https://api.example.org/v3/spaces/885735b5-aea4-4cf5-8e44-961af0e41920/features"
    },
    "organization": {
      "href": "https://api.example.org/v3/organizations/e00705b9-7b42-4561-ae97-2520399d2133"
    },
    "apply_manifest": {
      "href": "https://api.example.org/v3/spaces/885735b5-aea4-4cf5-8e44-961af0e41920/actions/apply_manifest",
      "method": "POST"
    }
  },
  "metadata": {
    "labels": {},
    "annotations": {}
  }
}`

const SubjectSpaceEmpty = `{
  "total_results": 0,
  "total_pages": 1,
  "prev_url": null,
  "next_url": null,
  "resources": []
}`

const SubjectSpaces = `{
   "resources": [
      {
         "guid": "guid-1",
         "type": "space_developer",
         "relationships": {
            "organization": {
               "data": {
                  "guid": "org-2-guid"
               }
            },
            "space": {
               "data": {
                  "guid": "space-1-guid"
               }
            }
         }
      },
      {
         "guid": "guid-2",
         "type": "space_developer",
         "relationships": {
            "organization": {
               "data": {
		  "guid": "org-2-guid"
               }
            },
            "space": {
               "data": {
                  "guid": "space-2-guid"
               }
            }
         }
      }
   ]
}`

const SubjectSpace1 = `{
   "resources": [
      {
         "guid": "guid-1",
         "type": "space_developer",
         "relationships": {
            "organization": {
               "data": {
                  "guid": "org-2-guid"
               }
            },
            "space": {
               "data": {
                  "guid": "space-1-guid"
               }
            }
         }
      }
  ]
}`

const SubjectSpace2 = `{}`

const SubjectSpacesPage1 = `{
   "pagination": {
      "total_results": 3,
      "total_pages": 3,
      "first": {
         "href": "https://api.example.org/v3/roles?page=1&per_page=2"
      },
      "last": {
         "href": "https://api.example.org/v3/roles?page=3&per_page=2"
      },
      "next": {
         "href": "https://api.example.org/v3/roles?page=2&per_page=2"
      },
      "previous": null
   },
   "resources": [
      {
         "guid": "40557c70-d1bd-4976-a2ab-a85f5e882418",
         "created_at": "2019-10-10T17:19:12Z",
         "updated_at": "2019-10-10T17:19:12Z",
         "type": "organization_auditor",
         "relationships": {
            "user": {
               "data": {
                  "guid": "59eadb5f-fc13-414f-84ba-77a35e239cc8"
               }
            },
            "organization": {
               "data": {
                  "guid": "05c5da3b-6cbc-421c-87c3-20bb3c41ab7c"
               }
            },
            "space": {
               "data": {
                  "guid": "space-1-guid"
               }
            }
         },
         "links": {
            "self": {
               "href": "https://api.example.org/v3/roles/40557c70-d1bd-4976-a2ab-a85f5e882418"
            },
            "user": {
               "href": "https://api.example.org/v3/users/59eadb5f-fc13-414f-84ba-77a35e239cc8"
            },
            "organization": {
               "href": "https://api.example.org/v3/organizations/05c5da3b-6cbc-421c-87c3-20bb3c41ab7c"
            }
         }
      }
   ]
}`
const SubjectSpacesPage2 = `{
   "pagination": {
      "total_results": 3,
      "total_pages": 3,
      "first": {
         "href": "https://api.example.org/v3/roles?page=1&per_page=2"
      },
      "last": {
         "href": "https://api.example.org/v3/roles?page=3&per_page=2"
      },
      "next": {
         "href": "https://api.example.org/v3/roles?page=3&per_page=2"
      },
      "previous": {
         "href": "https://api.example.org/v3/roles?page=1&per_page=2"
      }
   },
   "resources": [
      {
         "guid": "40557c70-d1bd-4976-a2ab-a85f5e882418",
         "created_at": "2019-10-10T17:19:12Z",
         "updated_at": "2019-10-10T17:19:12Z",
         "type": "organization_auditor",
         "relationships": {
            "user": {
               "data": {
                  "guid": "59eadb5f-fc13-414f-84ba-77a35e239cc8"
               }
            },
            "organization": {
               "data": {
                  "guid": "05c5da3b-6cbc-421c-87c3-20bb3c41ab7c"
               }
            },
            "space": {
               "data": {
                  "guid": "space-2-guid"
               }

            }
         },
         "links": {
            "self": {
               "href": "https://api.example.org/v3/roles/40557c70-d1bd-4976-a2ab-a85f5e882418"
            },
            "user": {
               "href": "https://api.example.org/v3/users/59eadb5f-fc13-414f-84ba-77a35e239cc8"
            },
            "organization": {
               "href": "https://api.example.org/v3/organizations/05c5da3b-6cbc-421c-87c3-20bb3c41ab7c"
            }
         }
      }
   ]
}`
const SubjectSpacesPage3 = `{
   "pagination": {
      "total_results": 3,
      "total_pages": 3,
      "first": {
         "href": "https://api.example.org/v3/roles?page=1&per_page=2"
      },
      "last": {
         "href": "https://api.example.org/v3/roles?page=3&per_page=2"
      },
      "next": null,
      "previous": {
         "href": "https://api.example.org/v3/roles?page=2&per_page=2"
      }
   },
   "resources": [
      {
         "guid": "40557c70-d1bd-4976-a2ab-a85f5e882418",
         "created_at": "2019-10-10T17:19:12Z",
         "updated_at": "2019-10-10T17:19:12Z",
         "type": "organization_auditor",
         "relationships": {
            "user": {
               "data": {
                  "guid": "59eadb5f-fc13-414f-84ba-77a35e239cc8"
               }
            },
            "organization": {
               "data": {
                  "guid": "05c5da3b-6cbc-421c-87c3-20bb3c41ab7c"
               }
            },
            "space": {
               "data": {
                  "guid": "space-3-guid"
               }
            }
         },
         "links": {
            "self": {
               "href": "https://api.example.org/v3/roles/40557c70-d1bd-4976-a2ab-a85f5e882418"
            },
            "user": {
               "href": "https://api.example.org/v3/users/59eadb5f-fc13-414f-84ba-77a35e239cc8"
            },
            "organization": {
               "href": "https://api.example.org/v3/organizations/05c5da3b-6cbc-421c-87c3-20bb3c41ab7c"
            }
         }
      }
   ]
}`
