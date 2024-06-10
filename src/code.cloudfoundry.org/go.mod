module code.cloudfoundry.org

go 1.21

toolchain go1.21.6

replace (
	example-apps/spammer => ../example-apps/spammer
	github.com/containernetworking/cni => github.com/containernetworking/cni v1.1.2
	github.com/containernetworking/plugins => github.com/containernetworking/plugins v1.1.1

	github.com/nats-io/gnatsd => github.com/nats-io/gnatsd v1.1.1-0.20180411231007-da89364d9d43
	github.com/nats-io/go-nats => github.com/nats-io/go-nats v1.5.1-0.20180331191609-247b2a84d8d0
	github.com/nats-io/nats-top => github.com/nats-io/nats-top v0.3.3-0.20160824043733-1c2a6920a922
)

require (
	code.cloudfoundry.org/bbs v0.0.0-20240521125508-20d3971ce31b
	code.cloudfoundry.org/cf-networking-helpers v0.0.0-20240606171834-15498a222c4a
	code.cloudfoundry.org/clock v1.1.0
	code.cloudfoundry.org/debugserver v0.0.0-20240605172147-3433a40ea1bc
	code.cloudfoundry.org/filelock v0.0.0-20240605152813-891eacc2974f
	code.cloudfoundry.org/garden v0.0.0-20240604193611-1a2f0f635229
	code.cloudfoundry.org/lager/v3 v3.0.3
	code.cloudfoundry.org/locket v0.0.0-20240521151413-b344fdd15d03
	code.cloudfoundry.org/policy_client v0.0.0-20240607171900-290f353c511b
	code.cloudfoundry.org/tlsconfig v0.0.0-20240606172222-82aa02bc07ea
	example-apps/spammer v0.0.0-00010101000000-000000000000
	github.com/benjamintf1/unmarshalledmatchers v1.0.0
	github.com/cf-container-networking/sql-migrate v0.0.0-20191108002617-83f2bdabdc5d
	github.com/cloudfoundry-community/go-uaa v0.3.2
	github.com/cloudfoundry/cf-test-helpers/v2 v2.9.0
	github.com/cloudfoundry/dropsonde v1.1.0
	github.com/containernetworking/cni v1.2.0
	github.com/containernetworking/plugins v1.5.0
	github.com/coreos/go-iptables v0.7.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/montanaflynn/stats v0.7.1
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v1.8.1
	github.com/nats-io/nats-top v0.6.1
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.19.0
	github.com/onsi/gomega v1.33.1
	github.com/pivotal-cf-experimental/gomegamatchers v0.0.0-20180326192815-e36bfcc98c3a
	github.com/pivotal-cf-experimental/rainmaker v0.0.0-20160401052143-d533d01b7c52
	github.com/pivotal-cf/paraphernalia v0.0.0-20180203224945-a64ae2051c20
	github.com/pkg/errors v0.9.1
	github.com/st3v/glager v0.4.0
	github.com/tedsuo/ifrit v0.0.0-20230516164442-7862c310ad26
	github.com/tedsuo/rata v1.0.0
	golang.org/x/net v0.26.0
	golang.org/x/sys v0.21.0
	gopkg.in/validator.v2 v2.0.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	code.cloudfoundry.org/diego-logging-client v0.0.0-20240607172319-a6c635476209 // indirect
	code.cloudfoundry.org/durationjson v0.0.0-20240605172149-1c08fce07291 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20240604201846-c756bfed2ed3 // indirect
	code.cloudfoundry.org/go-loggregator/v9 v9.2.1 // indirect
	code.cloudfoundry.org/inigo v0.0.0-20230228171622-18bab030e953 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20240515174134-adba8bce1248 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20240528025155-186aa0362fba // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/square/certstrap v1.3.0 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.step.sm/crypto v0.47.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240610135401-a8a62080eff3 // indirect
	google.golang.org/grpc v1.64.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
