module code.cloudfoundry.org

go 1.26.2

replace (
	code.cloudfoundry.org/locket => code.cloudfoundry.org/locket v0.0.0-20260602143356-23bea5865010
	example-apps/spammer => ../example-apps/spammer

	github.com/nats-io/go-nats => github.com/nats-io/go-nats v1.5.1-0.20180331191609-247b2a84d8d0
)

require (
	code.cloudfoundry.org/cf-networking-helpers v0.91.0
	code.cloudfoundry.org/clock v1.76.0
	code.cloudfoundry.org/debugserver v0.103.0
	code.cloudfoundry.org/diego-logging-client v0.113.0
	code.cloudfoundry.org/filelock v0.70.0
	code.cloudfoundry.org/garden v0.0.0-20260624020405-8f632509f05f
	code.cloudfoundry.org/lager/v3 v3.75.0
	code.cloudfoundry.org/locket v1.3.0
	code.cloudfoundry.org/policy_client v0.109.0
	code.cloudfoundry.org/tlsconfig v0.60.0
	example-apps/spammer v0.0.0-00010101000000-000000000000
	github.com/benjamintf1/unmarshalledmatchers v1.0.0
	github.com/cloudfoundry-community/go-uaa v0.4.0
	github.com/cloudfoundry/cf-acceptance-tests v1.9.1-0.20250312160631-048ab2ea8caa
	github.com/cloudfoundry/cf-test-helpers/v2 v2.13.0
	github.com/cloudfoundry/dropsonde v1.1.0
	github.com/containernetworking/cni v1.3.0
	github.com/containernetworking/plugins v1.9.1
	github.com/coreos/go-iptables v0.8.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/montanaflynn/stats v0.9.0
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v1.8.1
	github.com/nats-io/nats-server/v2 v2.14.2
	github.com/nats-io/nats-top v0.6.4
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.32.0
	github.com/onsi/gomega v1.42.1
	github.com/pivotal-cf-experimental/gomegamatchers v0.0.0-20180326192815-e36bfcc98c3a
	github.com/pivotal-cf-experimental/rainmaker v0.0.0-20160401052143-d533d01b7c52
	github.com/pivotal-cf/paraphernalia v0.0.0-20180203224945-a64ae2051c20
	github.com/pkg/errors v0.9.1
	github.com/rubenv/sql-migrate v1.8.1
	github.com/st3v/glager v0.4.0
	github.com/tedsuo/ifrit v0.0.0-20260418191334-846868129986
	github.com/tedsuo/rata v1.0.0
	golang.org/x/net v0.56.0
	golang.org/x/sys v0.46.0
	gopkg.in/validator.v2 v2.0.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	code.cloudfoundry.org/bbs v1.11.0 // indirect
	code.cloudfoundry.org/diego-db-helpers v0.5.0 // indirect
	code.cloudfoundry.org/durationjson v0.78.0 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20260622134745-74c0e1643bdd // indirect
	code.cloudfoundry.org/go-log-cache/v3 v3.1.2 // indirect
	code.cloudfoundry.org/go-loggregator/v10 v10.3.1 // indirect
	code.cloudfoundry.org/go-loggregator/v9 v9.2.1 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.7.2 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20260622134720-d7b012c5b9c4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-sql-driver/mysql v1.10.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/pprof v0.0.0-20260604005048-7023385849c0 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/lib/pq v1.12.3 // indirect
	github.com/minio/highwayhash v1.0.4 // indirect
	github.com/nats-io/jwt/v2 v2.8.2 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/square/certstrap v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	go.step.sm/crypto v0.84.1 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260622175928-b703f567277d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260622175928-b703f567277d // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
