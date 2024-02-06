module registry

replace github.com/codegangsta/cli => github.com/urfave/cli v1.19.0

replace github.com/Sirupsen/logrus => github.com/Sirupsen/logrus v0.8.7

replace github.com/amalgam8/amalgam8 => github.com/mariash/amalgam8 v0.3.1-0.20211124211225-20d5efebdebc

replace github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go v2.3.0+incompatible

go 1.20

require (
	github.com/Sirupsen/logrus v1.9.3
	github.com/amalgam8/amalgam8 v1.1.0
	github.com/codegangsta/cli v1.22.14
)

require (
	github.com/ant0ine/go-json-rest v3.3.2+incompatible // indirect
	github.com/garyburd/redigo v1.6.4 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/nicksnyder/go-i18n v1.10.3 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
