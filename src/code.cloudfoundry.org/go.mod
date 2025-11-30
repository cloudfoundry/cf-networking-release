module code.cloudfoundry.org

go 1.24.2

replace (
	example-apps/spammer => ../example-apps/spammer

	github.com/nats-io/go-nats => github.com/nats-io/go-nats v1.5.1-0.20180331191609-247b2a84d8d0
)

require (
	code.cloudfoundry.org/bbs v0.0.0-20251117222647-2ba9ab4f8077
	code.cloudfoundry.org/cf-networking-helpers v0.67.0
	code.cloudfoundry.org/clock v1.55.0
	code.cloudfoundry.org/debugserver v0.77.0
	code.cloudfoundry.org/diego-logging-client v0.81.0
	code.cloudfoundry.org/filelock v0.52.0
	code.cloudfoundry.org/garden v0.0.0-20251119022154-f0775181931d
	code.cloudfoundry.org/lager/v3 v3.55.0
	code.cloudfoundry.org/locket v0.0.0-20251117222557-be612341b29d
	code.cloudfoundry.org/policy_client v0.80.0
	code.cloudfoundry.org/tlsconfig v0.41.0
	example-apps/spammer v0.0.0-00010101000000-000000000000
	github.com/benjamintf1/unmarshalledmatchers v1.0.0
	github.com/cloudfoundry-community/go-uaa v0.3.5
	github.com/cloudfoundry/cf-acceptance-tests v1.9.1-0.20250312160631-048ab2ea8caa
	github.com/cloudfoundry/cf-test-helpers/v2 v2.13.0
	github.com/cloudfoundry/dropsonde v1.1.0
	github.com/containernetworking/cni v1.3.0
	github.com/containernetworking/plugins v1.8.0
	github.com/coreos/go-iptables v0.8.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/montanaflynn/stats v0.7.1
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v1.8.1
	github.com/nats-io/nats-server/v2 v2.12.2
	github.com/nats-io/nats-top v0.6.3
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.27.2
	github.com/onsi/gomega v1.38.2
	github.com/pivotal-cf-experimental/gomegamatchers v0.0.0-20180326192815-e36bfcc98c3a
	github.com/pivotal-cf-experimental/rainmaker v0.0.0-20160401052143-d533d01b7c52
	github.com/pivotal-cf/paraphernalia v0.0.0-20180203224945-a64ae2051c20
	github.com/pkg/errors v0.9.1
	github.com/rubenv/sql-migrate v1.8.1
	github.com/st3v/glager v0.4.0
	github.com/tedsuo/ifrit v0.0.0-20230516164442-7862c310ad26
	github.com/tedsuo/rata v1.0.0
	golang.org/x/net v0.47.0
	golang.org/x/sys v0.38.0
	gopkg.in/validator.v2 v2.0.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	code.cloudfoundry.org/durationjson v0.58.0 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20251124091730-becfb70ab838 // indirect
	code.cloudfoundry.org/go-log-cache/v3 v3.1.1 // indirect
	code.cloudfoundry.org/go-loggregator/v10 v10.2.0 // indirect
	code.cloudfoundry.org/go-loggregator/v9 v9.2.1 // indirect
	code.cloudfoundry.org/inigo v0.0.0-20230228171622-18bab030e953 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.5.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20251124090431-33e3494ff82b // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-tpm v0.9.7 // indirect
	github.com/google/pprof v0.0.0-20251114195745-4902fdda35c8 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/minio/highwayhash v1.0.4-0.20251030100505-070ab1a87a76 // indirect
	github.com/nats-io/jwt/v2 v2.8.0 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/square/certstrap v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	go.step.sm/crypto v0.74.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/oauth2 v0.33.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251124214823-79d6a2a48846 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251124214823-79d6a2a48846 // indirect
	google.golang.org/grpc v1.77.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
