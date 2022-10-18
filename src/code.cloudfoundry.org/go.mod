module code.cloudfoundry.org

replace github.com/nats-io/go-nats => github.com/nats-io/go-nats v1.5.1-0.20180331191609-247b2a84d8d0

replace github.com/containernetworking/cni => github.com/containernetworking/cni v1.1.2

replace github.com/containernetworking/plugins => github.com/containernetworking/plugins v1.1.1

replace github.com/nats-io/gnatsd => github.com/nats-io/gnatsd v1.1.1-0.20180411231007-da89364d9d43

replace github.com/nats-io/nats-top => github.com/nats-io/nats-top v0.3.3-0.20160824043733-1c2a6920a922

go 1.18

require (
	code.cloudfoundry.org/bbs v0.0.0-20220928214406-6d44a4df0134
	code.cloudfoundry.org/cf-networking-helpers v0.0.0-20221018152553-fd5c790ce598
	code.cloudfoundry.org/clock v1.0.0
	code.cloudfoundry.org/debugserver v0.0.0-20211123175613-a7ac7ce093eb
	code.cloudfoundry.org/diego-logging-client v0.0.0-20211220190808-bd0d93324d64 // indirect
	code.cloudfoundry.org/durationjson v0.0.0-20211123184609-ead4881606b1 // indirect
	code.cloudfoundry.org/filelock v0.0.0-20180314203404-13cd41364639
	code.cloudfoundry.org/garden v0.0.0-20210813150702-ba711ea09ea2
	code.cloudfoundry.org/go-diodes v0.0.0-20220325013804-800fb6f70e2f // indirect
	code.cloudfoundry.org/inigo v0.0.0-20221017175908-16defe1ae4df // indirect
	code.cloudfoundry.org/lager v2.0.0+incompatible
	code.cloudfoundry.org/locket v0.0.0-20220920165427-933b89909aba
	code.cloudfoundry.org/tlsconfig v0.0.0-20220621140725-0e6fbd869921
	example-apps/spammer v0.0.0-00010101000000-000000000000
	github.com/benjamintf1/unmarshalledmatchers v1.0.0
	github.com/cf-container-networking/sql-migrate v0.0.0-20191108002617-83f2bdabdc5d
	github.com/cloudfoundry-incubator/cf-test-helpers v1.0.0
	github.com/cloudfoundry/dropsonde v1.0.0
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.0.0
	github.com/coreos/go-iptables v0.6.0
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/go-test/deep v1.0.8 // indirect
	github.com/gofrs/uuid v4.2.0+incompatible // indirect
	github.com/golang/protobuf v1.5.2
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible // indirect
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.7 // indirect
	github.com/montanaflynn/stats v0.6.6
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v0.0.0-00010101000000-000000000000
	github.com/nats-io/nats-top v0.4.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.22.1
	github.com/pivotal-cf-experimental/gomegamatchers v0.0.0-20180326192815-e36bfcc98c3a
	github.com/pivotal-cf-experimental/rainmaker v0.0.0-20160401052143-d533d01b7c52
	github.com/pivotal-cf-experimental/warrant v0.0.0-20211122194707-17385443920f
	github.com/pivotal-cf/paraphernalia v0.0.0-20180203224945-a64ae2051c20
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/st3v/glager v0.3.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	github.com/tedsuo/rata v1.0.0
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/crypto v0.0.0-20211215153901-e495a2d5b3d3 // indirect
	golang.org/x/net v0.0.0-20221017152216-f25eb7ecb193
	golang.org/x/sys v0.1.0
	google.golang.org/genproto v0.0.0-20221014213838-99cd37c6964a // indirect
	google.golang.org/grpc v1.50.1
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/validator.v2 v2.0.0-20210331031555-b37d688a7fb0
	gopkg.in/yaml.v2 v2.4.0
)

require code.cloudfoundry.org/policy_client v0.0.0-20220509212643-31108c669266

require (
	code.cloudfoundry.org/go-loggregator/v8 v8.0.5 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/gosteno v0.0.0-20150423193413-0c8581caea35 // indirect
	github.com/cloudfoundry/sonde-go v0.0.0-20220627221915-ff36de9c3435 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/square/certstrap v1.2.0 // indirect
	github.com/stretchr/testify v1.8.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace example-apps/spammer => ../example-apps/spammer
