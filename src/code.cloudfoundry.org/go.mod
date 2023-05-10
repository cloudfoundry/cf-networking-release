module code.cloudfoundry.org

go 1.20

replace (
	example-apps/spammer => ../example-apps/spammer
	github.com/containernetworking/cni => github.com/containernetworking/cni v1.1.2
	github.com/containernetworking/plugins => github.com/containernetworking/plugins v1.1.1

	github.com/nats-io/gnatsd => github.com/nats-io/gnatsd v1.1.1-0.20180411231007-da89364d9d43
	github.com/nats-io/go-nats => github.com/nats-io/go-nats v1.5.1-0.20180331191609-247b2a84d8d0
	github.com/nats-io/nats-top => github.com/nats-io/nats-top v0.3.3-0.20160824043733-1c2a6920a922

	// Needed until https://github.com/st3v/glager/pull/6/files is merged
	github.com/st3v/glager v0.3.0 => github.com/geofffranks/glager v0.0.0-20230329153253-21ef5c265920

	// Prevents test failures in bosh-dns-adapter when grpc is upgraded
	google.golang.org/grpc => google.golang.org/grpc v1.50.1
)

require (
	code.cloudfoundry.org/bbs v0.0.0-20230419194333-54391379d38f
	code.cloudfoundry.org/cf-networking-helpers v0.0.0-20230510133512-ebb4c931f5d5
	code.cloudfoundry.org/cf-test-helpers v1.0.0
	code.cloudfoundry.org/clock v1.1.0
	code.cloudfoundry.org/debugserver v0.0.0-20230508035730-c4fc5f67e21e
	code.cloudfoundry.org/filelock v0.0.0-20230410204127-470838d066c5
	code.cloudfoundry.org/garden v0.0.0-20230509175005-ab96b4719f65
	code.cloudfoundry.org/lager/v3 v3.0.1
	code.cloudfoundry.org/locket v0.0.0-20230424201354-6cd5416498b6
	code.cloudfoundry.org/policy_client v0.0.0-20230405194717-ac8a054e5f69
	code.cloudfoundry.org/tlsconfig v0.0.0-20230320190829-8f91c367795b
	example-apps/spammer v0.0.0-00010101000000-000000000000
	github.com/benjamintf1/unmarshalledmatchers v1.0.0
	github.com/cf-container-networking/sql-migrate v0.0.0-20191108002617-83f2bdabdc5d
	github.com/cloudfoundry/cf-test-helpers/v2 v2.5.0
	github.com/cloudfoundry/dropsonde v1.0.1-0.20230324134055-c6dd7c5e990e
	github.com/containernetworking/cni v1.1.2
	github.com/containernetworking/plugins v1.3.0
	github.com/coreos/go-iptables v0.6.0
	github.com/golang/protobuf v1.5.3
	github.com/jmoiron/sqlx v1.3.5
	github.com/montanaflynn/stats v0.7.0
	github.com/nats-io/gnatsd v0.0.0-00010101000000-000000000000
	github.com/nats-io/go-nats v1.8.1
	github.com/nats-io/nats-top v0.0.0-00010101000000-000000000000
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo/v2 v2.9.4
	github.com/onsi/gomega v1.27.6
	github.com/pivotal-cf-experimental/gomegamatchers v0.0.0-20180326192815-e36bfcc98c3a
	github.com/pivotal-cf-experimental/rainmaker v0.0.0-20160401052143-d533d01b7c52
	github.com/pivotal-cf-experimental/warrant v0.0.0-20211122194707-17385443920f
	github.com/pivotal-cf/paraphernalia v0.0.0-20180203224945-a64ae2051c20
	github.com/pkg/errors v0.9.1
	github.com/st3v/glager v0.3.0
	github.com/tedsuo/ifrit v0.0.0-20230330192023-5cba443a66c4
	github.com/tedsuo/rata v1.0.0
	golang.org/x/net v0.10.0
	golang.org/x/sys v0.8.0
	google.golang.org/grpc v1.55.0
	gopkg.in/validator.v2 v2.0.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	code.cloudfoundry.org/diego-logging-client v0.0.0-20230508200927-824f04190d59 // indirect
	code.cloudfoundry.org/durationjson v0.0.0-20230508215707-5acd8d1a9dec // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20230508203442-8ce2048f62dc // indirect
	code.cloudfoundry.org/go-loggregator/v8 v8.0.5 // indirect
	code.cloudfoundry.org/inigo v0.0.0-20230228171622-18bab030e953 // indirect
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry-incubator/cf-test-helpers v1.0.0 // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20230412182205-eaf74d09b55a // indirect
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/go-test/deep v1.1.0 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20230510103437-eeec1cb781c3 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/square/certstrap v1.3.0 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.step.sm/crypto v0.28.0 // indirect
	golang.org/x/crypto v0.9.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
