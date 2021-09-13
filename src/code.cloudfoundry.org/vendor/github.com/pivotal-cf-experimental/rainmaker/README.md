# A Client Library for CC, written in Go
Rainmaker provides a library of functionality for interacting with the Cloud Controller.
The library supports management of organizations, spaces and users.

[![GoDoc](https://godoc.org/github.com/pivotal-cf-experimental/rainmaker?status.svg)](https://godoc.org/github.com/pivotal-cf-experimental/rainmaker)

## Caveat Emptor
Rainmaker is still under development and the APIs shown herein are subject to change.

## Example

Rainmaker can be used in a variety of ways. Here is a simple example to get you started:

```go
package main

import (
	"log"

	"github.com/pivotal-cf-experimental/rainmaker"
)

func main() {
	r := rainmaker.NewClient(rainmaker.Config{
		Host: "https://api.example.com",
	})

	org, err := client.Organizations.Create("A New Org", token)
	if err != nil {
		log.Fatalf("Unable to create organization: %s", err)
	}

	fetchedOrg, err := client.Organizations.Get(org.GUID, token)
	if err != nil {
		log.Fatalf("Unable to fetch organization: %s", err)
	}

	log.Printf("%+v\n", fetchedOrg)
	// => {GUID:eedacbb8-72c6-11e5-a4a4-6b0a4f4c3afa, Name:A New Org, ...}

	space, err := client.Spaces.Create("Interesting Space", org.GUID, token)
	if err != nil {
		log.Fatalf("Unable to create space: %s", err)
	}

	fetchedSpace, err := client.Spaces.Get(space.GUID, token)
	if err != nil {
		log.Fatalf("Unable to fetch space: %s", err)
	}

	log.Printf("%+v\n", fetchedSpace)
	// => {GUID:7aa59fd2-72c8-11e5-b644-4f220f7a6847, OrganizationGUID:eedacbb8-72c6-11e5-a4a4-6b0a4f4c3afa, Name:Interesting Space, ...}

	spaceList, err := client.Spaces.List(token)
	if err != nil {
		log.Fatalf("Unable to fetch the list of spaces: %s", err)
	}

	log.Printf("%+v\n", spaceList.Spaces)
	// => [{GUID:7aa59fd2-72c8-11e5-b644-4f220f7a6847, OrganizationGUID:eedacbb8-72c6-11e5-a4a4-6b0a4f4c3afa, Name:Interesting Space, ...}]

	err = client.Spaces.Delete(space.GUID, token)
	if err != nil {
		log.Fatalf("Unable to delete organization: %s", err)
	}

	err = client.Organizations.Delete(org.GUID, token)
	if err != nil {
		log.Fatalf("Unable to delete organization: %s", err)
	}
}
```
