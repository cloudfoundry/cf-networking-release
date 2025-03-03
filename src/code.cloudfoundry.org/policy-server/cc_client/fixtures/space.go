package fixtures

const Space = `{
  "resources": [
    {
      "guid": "some-space-guid",
      "name": "some-space-name",
      "relationships": {
        "organization": {
          "data": {
            "guid": "6e1ca5aa-55f1-4110-a97f-1f3473e771b9"
          }
        }
      }
    }
  ]
}`

const Space1 = `{
  "resources": [
    {
      "guid": "space-1-guid",
      "name": "space-1",
      "relationships": {
        "organization": {
          "data": {
            "guid": "org-1-guid"
          }
        }
      }
    }
  ]
}`
const Space2 = `{
  "resources": [
    {
      "guid": "space-2-guid",
      "name": "space-2",
      "relationships": {
        "organization": {
          "data": {
            "guid": "org-1-guid"
          }
        }
      }
    }
  ]
}`
