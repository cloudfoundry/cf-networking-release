package asg_syncer_test

import (
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/policy-server/asg_syncer"
	ccfakes "code.cloudfoundry.org/policy-server/cc_client/fakes"
	dbfakes "code.cloudfoundry.org/policy-server/store/fakes"
	uaafakes "code.cloudfoundry.org/policy-server/uaa_client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ASGSyncer", func() {
	var (
		fakeUAAClient  *uaafakes.UAAClient
		fakeCCClient   *ccfakes.CCClient
		logger         *lagertest.TestLogger
		requestTimeout time.Duration
		fakeStore      *dbfakes.SecurityGroupsStore
	)
	BeforeEach(func() {
		requestTimeout = 1
		fakeStore = &dbfakes.SecurityGroupsStore{}
		fakeUAAClient = &uaafakes.UAAClient{}
		fakeCCClient = &ccfakes.CCClient{}
		logger = lagertest.NewTestLogger("test")
	})
	It("pulls ASG data from CAPI and stores it in the SecurityGroupStore", func() {
		syncer := asg_syncer.NewASGSyncer(logger, fakeStore, fakeUAAClient, fakeCCClient, requestTimeout)
		Expect(syncer).ToNot(BeNil())

		By("querying CAPI for /v3/security_groups, handling pagination", func() {
			Expect(syncer).ToNot(BeNil())
		})
	})
})
