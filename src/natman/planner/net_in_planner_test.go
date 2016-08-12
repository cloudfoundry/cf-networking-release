package planner_test

import (
	"errors"
	"lib/rules"
	"natman/planner"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/gardenfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetInPlanner", func() {
	var (
		p              *planner.NetInPlanner
		fakeClient     *gardenfakes.FakeClient
		fakeContainer1 *gardenfakes.FakeContainer
		fakeContainer2 *gardenfakes.FakeContainer
	)

	BeforeEach(func() {
		fakeContainer1 = &gardenfakes.FakeContainer{}
		fakeContainer2 = &gardenfakes.FakeContainer{}
		fakeClient = &gardenfakes.FakeClient{}
		p = &planner.NetInPlanner{
			GardenClient: fakeClient,
		}
	})

	Describe("GetRules", func() {
		BeforeEach(func() {
			fakeContainer1.InfoReturns(garden.ContainerInfo{
				ContainerIP: "10.255.1.2",
				ExternalIP:  "10.254.16.2",
				Properties:  map[string]string{"network.app_id": "some-app-guid"},
				MappedPorts: []garden.PortMapping{{
					HostPort:      1234,
					ContainerPort: 8080,
				}, {
					HostPort:      6789,
					ContainerPort: 2222,
				}},
			}, nil)

			fakeContainer2.InfoReturns(garden.ContainerInfo{
				ContainerIP: "10.255.1.3",
				ExternalIP:  "10.254.16.2",
				Properties:  map[string]string{"network.app_id": "some-other-app-guid"},
				MappedPorts: []garden.PortMapping{{
					HostPort:      1234,
					ContainerPort: 8080,
				}},
			}, nil)

			fakeClient.ContainersReturns([]garden.Container{
				fakeContainer1,
				fakeContainer2,
			}, nil)
		})

		It("contacts the gets all of the containers from the garden server", func() {
			_, err := p.GetRules()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClient.ContainersCallCount()).To(Equal(1))
			Expect(fakeClient.ContainersArgsForCall(0)).To(Equal(garden.Properties{}))
		})
		It("returns a list of rules", func() {
			r, err := p.GetRules()
			Expect(err).NotTo(HaveOccurred())
			Expect(r).To(ConsistOf([]rules.GenericRule{{
				Properties: []string{
					"-d", "10.254.16.2",
					"-p", "tcp",
					"-m", "tcp", "--dport", "1234",
					"--jump", "DNAT",
					"--to-destination", "10.255.1.2:8080",
					"-m", "comment", "--comment", "dst:some-app-guid",
				},
			}, {
				Properties: []string{
					"-d", "10.254.16.2",
					"-p", "tcp",
					"-m", "tcp", "--dport", "6789",
					"--jump", "DNAT",
					"--to-destination", "10.255.1.2:2222",
					"-m", "comment", "--comment", "dst:some-app-guid",
				},
			}, {
				Properties: []string{
					"-d", "10.254.16.2",
					"-p", "tcp",
					"-m", "tcp", "--dport", "1234",
					"--jump", "DNAT",
					"--to-destination", "10.255.1.3:8080",
					"-m", "comment", "--comment", "dst:some-other-app-guid",
				},
			}}))
		})

		Context("when getting the containers fails", func() {
			BeforeEach(func() {
				fakeClient.ContainersReturns(nil, errors.New("banana"))
			})
			It("returns the error", func() {
				_, err := p.GetRules()
				Expect(err).To(MatchError("banana"))
			})
		})

		Context("when getting the container info fails", func() {
			BeforeEach(func() {
				fakeContainer1.InfoReturns(garden.ContainerInfo{}, errors.New("potato"))
			})
			It("returns the error", func() {
				_, err := p.GetRules()
				Expect(err).To(MatchError("potato"))
			})
		})
	})
})
