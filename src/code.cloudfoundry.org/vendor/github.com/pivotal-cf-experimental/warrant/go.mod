module github.com/pivotal-cf-experimental/warrant

go 1.12

require (
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.4.1
	golang.org/x/net v0.0.0-20180808004115-f9ce57c11b24 // indirect
	golang.org/x/text v0.3.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.2.1 // indirect
)

replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
