# A Client Library for UAA, written in Go
Warrant provides a library of functionality for interacting with the UAA service.
The library supports management of users, clients, groups and tokens.

[![GoDoc](https://godoc.org/github.com/pivotal-cf-experimental/warrant?status.svg)](https://godoc.org/github.com/pivotal-cf-experimental/warrant)

## Caveat Emptor
Warrant is still under development and the APIs shown herein are subject to change.

## Example

Warrant can be used in a variety of ways. Here is a simple example to get you started:

```go
package main

import (
	"log"

	"github.com/pivotal-cf-experimental/warrant"
)

func main() {
	w := warrant.New(warrant.Config{
		Host: "https://uaa.example.com",
	})

	clientToken, err := w.Clients.GetToken("admin", "admin-secret")
	if err != nil {
		log.Fatalf("Unable to fetch client token: %s", err)
	}

	user, err := w.Users.Create("my-user", "me@example.com", clientToken)
	if err != nil {
		log.Fatalf("Unable to create user: %s", err)
	}

	err = w.Users.SetPassword(user.ID, "my-password", clientToken)
	if err != nil {
		log.Fatalf("Unable to set user password: %s", err)
	}

	userToken, err := w.Users.GetToken("my-user", "my-password")
	if err != nil {
		log.Fatalf("Unable to fetch user token: %s", err)
	}

	decodedToken, err := w.Tokens.Decode(userToken)
	if err != nil {
		log.Fatalf("Unable to decode user token: %s", err)
	}

	log.Printf("%+v\n", decodedToken)
	// => {ClientID:cf, UserID:80d4fd0b-119f-4fc7-a800-eb186bc8e766, Scopes:[openid, cloud_controller.read]}
}
```
