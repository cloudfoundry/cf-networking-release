package rule_updater_test

import (
	"errors"
	"netman-agent/fakes"
	"netman-agent/models"
	"netman-agent/rule_updater"

	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("RuleUpdater", func() {
	var (
		ruleUpdater  *rule_updater.Updater
		storeReader  *fakes.StoreReader
		policyClient *fakes.PolicyClient
		logger       *lagertest.TestLogger
	)

	BeforeEach(func() {
		storeReader = &fakes.StoreReader{}
		policyClient = &fakes.PolicyClient{}
		logger = lagertest.NewTestLogger("test")

		storeReader.GetContainersReturns(models.Containers{
			"some-app-guid": []models.Container{{
				ID: "some-container-id",
				IP: "8.8.8.8",
			}},
			"some-other-app-guid": []models.Container{{
				ID: "some-other-container-id",
				IP: "8.8.8.9",
			}},
		})

		policyClient.GetPoliciesReturns([]models.Policy{{
			models.Source{
				ID: "some-app-guid",
			},
			models.Destination{
				ID:       "some-other-app-guid",
				Port:     5555,
				Protocol: "tcp",
			},
		}}, nil)

		ruleUpdater = rule_updater.New(logger, storeReader, policyClient)
	})

	Describe("Update", func() {
		It("gets the policies and containers", func() {
			err := ruleUpdater.Update()
			Expect(err).NotTo(HaveOccurred())

			Expect(storeReader.GetContainersCallCount()).To(Equal(1))
			Expect(policyClient.GetPoliciesCallCount()).To(Equal(1))
		})

		It("logs the rules it is about to enforce", func() {
			err := ruleUpdater.Update()
			Expect(err).NotTo(HaveOccurred())

			Expect(logger).To(gbytes.Say(`enforce-local-rule.*{"dstIP":"8.8.8.9","port":5555,"proto":"tcp","srcIP":"8.8.8.8"}`))
		})

		Context("when the policy client fails", func() {
			BeforeEach(func() {
				policyClient.GetPoliciesReturns(nil, errors.New("banana"))
			})
			It("returns and logs the error", func() {
				err := ruleUpdater.Update()
				Expect(err).To(MatchError("get policies failed: banana"))

				Expect(logger).To(gbytes.Say(`get-policies.*banana`))
			})
		})
	})

})
