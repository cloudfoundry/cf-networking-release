package api_test

import (
	"policy-server/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Egress Validator", func() {
	var (
		validator      api.EgressValidator
		egressPolicies []api.EgressPolicy
	)

	BeforeEach(func() {
		validator = api.EgressValidator{}

		egressPolicies = []api.EgressPolicy{
			{
				Source: &api.EgressSource{
					ID: "source id",
				},
				Destination: &api.EgressDestination{
					IPRanges: []api.IPRange{
						{Start: "1.2.3.4", End: "5.6.7.8"},
					},
					Protocol: "tcp",
				},
			},
		}
	})

	Describe("ValidateEgressPolicies", func() {
		It("should not return an error when given a valid egress policy", func() {
			Expect(validator.ValidateEgressPolicies(egressPolicies)).To(Succeed())
		})

		It("requires a source", func() {
			egressPolicies[0].Source = nil

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("missing egress source"))
		})

		It("type must be app, space or empty", func() {
			egressPolicies[0].Source.Type = "invalid"

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("source type must be app or space"))

			for _, validType := range []string{"app", "space", ""} {
				egressPolicies[0].Source.Type = validType
				err := validator.ValidateEgressPolicies(egressPolicies)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("requires a source guid", func() {
			egressPolicies[0].Source.ID = ""

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("missing egress source ID"))
		})

		It("requires a destination", func() {
			egressPolicies[0].Destination = nil

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("missing egress destination"))
		})

		It("requires a destination protocol", func() {
			egressPolicies[0].Destination.Protocol = ""

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("missing egress destination protocol"))
		})

		It("requires protocol to be tcp, udp, or icmp", func() {
			egressPolicies[0].Destination.Protocol = "invalid"

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("protocol must be tcp, udp, or icmp"))
		})

		Context("when protocol is icmp", func() {
			BeforeEach(func() {
				egressPolicies[0].Destination.Protocol = "icmp"
			})

			It("requires icmp type", func() {
				err := validator.ValidateEgressPolicies(egressPolicies)
				Expect(err).To(MatchError("missing icmp type"))
			})

			It("requires icmp code", func() {
				i := 2
				egressPolicies[0].Destination.ICMPType = &i

				err := validator.ValidateEgressPolicies(egressPolicies)
				Expect(err).To(MatchError("missing icmp code"))
			})

			It("does not allow ports to be defined", func() {
				i := 2
				egressPolicies[0].Destination.ICMPType = &i
				egressPolicies[0].Destination.ICMPCode = &i
				egressPolicies[0].Destination.Ports = []api.Ports{
					{
						Start: 11,
						End:   12,
					},
				}

				err := validator.ValidateEgressPolicies(egressPolicies)
				Expect(err).To(MatchError("ports can not be defined with icmp"))
			})
		})

		It("requires ip range", func() {
			egressPolicies[0].Destination.IPRanges = []api.IPRange{}

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("expected exactly one iprange"))
		})

		It("only allows for one ip range", func() {
			egressPolicies[0].Destination.IPRanges = []api.IPRange{
				{Start: "1", End: "2"},
				{Start: "1", End: "2"},
			}

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("expected exactly one iprange"))
		})

		It("requires valid start v4 ip addresses", func() {
			egressPolicies[0].Destination.IPRanges[0].Start = "1"

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("invalid ipv4 start ip address for ip range: 1"))

			egressPolicies[0].Destination.IPRanges[0].Start = "2001:db8:85a3:0:0:8a2e:370:7334"

			err = validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("invalid ipv4 start ip address for ip range: 2001:db8:85a3:0:0:8a2e:370:7334"))
		})

		It("requires valid end v4 ip addresses", func() {
			egressPolicies[0].Destination.IPRanges[0].End = "255.255.255.256"

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("invalid ipv4 end ip address for ip range: 255.255.255.256"))

			egressPolicies[0].Destination.IPRanges[0].End = "2001:db8:85a3:0:0:8a2e:370:7334"

			err = validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("invalid ipv4 end ip address for ip range: 2001:db8:85a3:0:0:8a2e:370:7334"))
		})

		It("requires start ip address to be before end", func() {
			egressPolicies[0].Destination.IPRanges[0].Start = "1.2.3.4"
			egressPolicies[0].Destination.IPRanges[0].End = "1.2.3.3"

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("start ip address should be before end ip address: start: 1.2.3.4 end: 1.2.3.3"))
		})

		It("fails on first bad record", func() {
			egressPolicies = []api.EgressPolicy{
				{
					Source: &api.EgressSource{
						ID: "good-record",
					},
					Destination: &api.EgressDestination{
						IPRanges: []api.IPRange{
							{Start: "1.2.3.4", End: "5.6.7.8"},
						},
						Protocol: "tcp",
					},
				},
				{
					Source: &api.EgressSource{
						ID: "bad-record",
					},
					Destination: &api.EgressDestination{},
				},
				{
					Source: &api.EgressSource{
						ID: "another-good-record",
					},
					Destination: &api.EgressDestination{
						IPRanges: []api.IPRange{
							{Start: "1.2.3.4", End: "5.6.7.8"},
						},
						Protocol: "tcp",
					},
				},
			}

			err := validator.ValidateEgressPolicies(egressPolicies)
			Expect(err).To(MatchError("missing egress destination protocol"))
		})
	})
})
