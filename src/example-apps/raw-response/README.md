# raw-response

The `raw-response` app was designed to make writing arbitrary
HTTP responses out from a CloudFoundry app, bypassing any of
Golang's built-in net/http or net/httputil behaviors.


## How it works

It will read a request in, print it out to stdout, and then
respond with the data included in the file `output-data`.

If any errors are encountered, they'll be printed to stderr.


## How to use it

1. Create a file in this directory named `output-data`,
and put the exact response you returned by the app there.

1. Either `cf push` the app to CloudFoundry, or `go run .` to run locally.

1. curl the app, and watch it respond. When run locally it will default to port
   8080 (on all interfaces).
