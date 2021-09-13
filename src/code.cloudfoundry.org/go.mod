module code.cloudfoundry.org

replace github.com/nats-io/go-nats => github.com/nats-io/go-nats v1.5.1-0.20180331191609-247b2a84d8d0

replace github.com/containernetworking/cni => github.com/containernetworking/cni v0.6.0

replace github.com/containernetworking/plugins => github.com/containernetworking/plugins v0.6.0

replace github.com/square/certstrap => github.com/square/certstrap v1.1.1

replace github.com/pivotal-cf-experimental/warrant => github.com/pivotal-cf-experimental/warrant v0.0.0-20170102215731-9d632d130b6a

replace github.com/nats-io/gnatsd => github.com/nats-io/gnatsd v1.1.1-0.20180411231007-da89364d9d43

replace github.com/nats-io/nats-top => github.com/nats-io/nats-top v0.3.3-0.20160824043733-1c2a6920a922

go 1.16

require (
	code.cloudfoundry.org/bbs v0.0.0-20210727125654-2ad50317f7ed
	code.cloudfoundry.org/cf-networking-helpers v0.0.0-20210825141236-777da71209d5
	code.cloudfoundry.org/clock v1.0.0
	code.cloudfoundry.org/debugserver v0.0.0-20210608171006-d7658ce493f4
	code.cloudfoundry.org/filelock v0.0.0-20180314203404-13cd41364639
	code.cloudfoundry.org/garden v0.0.0-20210813150702-ba711ea09ea2
	code.cloudfoundry.org/lager v2.0.0+incompatible
	code.cloudfoundry.org/tlsconfig v0.0.0-20210615191307-5d92ef3894a7
	github.com/benjamintf1/unmarshalledmatchers v1.0.0
	github.com/cf-container-networking/sql-migrate v0.0.0-20191108002617-83f2bdabdc5d
	github.com/cloudfoundry-incubator/cf-test-helpers v1.0.0
	github.com/cloudfoundry/dropsonde v1.0.0
	github.com/cloudfoundry/gosteno v0.0.0-20150423193413-0c8581caea35 // indirect
	github.com/cloudfoundry/loggregatorlib v0.0.0-20170823162133-36eddf15ef12 // indirect
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.0.0
	github.com/coreos/go-iptables v0.6.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/jmoiron/sqlx v1.3.4
	github.com/lib/pq v1.10.2
	github.com/montanaflynn/stats v0.6.6
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v0.0.0-00010101000000-000000000000
	github.com/nats-io/nats-top v0.4.0
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/pivotal-cf-experimental/gomegamatchers v0.0.0-20180326192815-e36bfcc98c3a
	github.com/pivotal-cf-experimental/rainmaker v0.0.0-20160401052143-d533d01b7c52
	github.com/pivotal-cf-experimental/warrant v0.0.0-20190619140101-cb9c3921aa62
	github.com/pivotal-cf/paraphernalia v0.0.0-20180203224945-a64ae2051c20
	github.com/pkg/errors v0.9.1
	github.com/st3v/glager v0.3.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	github.com/tedsuo/rata v1.0.0
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e
	google.golang.org/grpc v1.40.0
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/validator.v2 v2.0.0-20210331031555-b37d688a7fb0
	gopkg.in/yaml.v2 v2.4.0
	launchpad.net/gocheck v0.0.0-20140225173054-000000000087 // indirect
)
