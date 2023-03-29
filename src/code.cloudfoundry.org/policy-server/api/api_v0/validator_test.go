package api_v0_test

import (
	"code.cloudfoundry.org/policy-server/api/api_v0"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PolicyValidator", func() {
	var validator api_v0.Validator

	BeforeEach(func() {
		validator = api_v0.Validator{}
	})

	Describe("ValidatePolicies", func() {
		It("does not error for valid policies", func() {
			policies := []api_v0.Policy{
				api_v0.Policy{
					Source: api_v0.Source{
						ID: "some-source-id",
					},
					Destination: api_v0.Destination{
						ID:       "some-destination-id",
						Protocol: "tcp",
						Port:     42,
					},
				},
			}

			err := validator.ValidatePolicies(policies)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the policies list is nil", func() {
			It("returns a useful error", func() {
				err := validator.ValidatePolicies(nil)
				Expect(err).To(MatchError("missing policies"))
			})
		})

		Context("when the policies list is empty", func() {
			It("returns a useful error", func() {
				err := validator.ValidatePolicies([]api_v0.Policy{})
				Expect(err).To(MatchError("missing policies"))
			})
		})

		Context("when source id is missing", func() {
			It("returns a useful error", func() {
				policies := []api_v0.Policy{
					api_v0.Policy{
						Source: api_v0.Source{
							ID:  "",
							Tag: "",
						},
						Destination: api_v0.Destination{
							ID:       "some-destination-id",
							Tag:      "",
							Protocol: "tcp",
							Port:     42,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("missing source id"))
			})
		})

		Context("when destination id is missing", func() {
			It("returns a useful error", func() {
				policies := []api_v0.Policy{
					api_v0.Policy{
						Source: api_v0.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: api_v0.Destination{
							ID:       "",
							Tag:      "",
							Protocol: "tcp",
							Port:     42,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("missing destination id"))
			})
		})

		Context("when invalid destination protocol", func() {
			It("returns a useful error", func() {
				policies := []api_v0.Policy{
					api_v0.Policy{
						Source: api_v0.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: api_v0.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "banana",
							Port:     42,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid destination protocol, specify either udp or tcp"))
			})
		})

		Context("when the port is less than or equal to 0", func() {
			It("returns a useful error", func() {
				policies := []api_v0.Policy{
					api_v0.Policy{
						Source: api_v0.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: api_v0.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Port:     -42,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid port -42, must be in range 1-65535"))
			})
		})

		Context("when the port is 0", func() {
			It("returns a useful error", func() {
				policies := []api_v0.Policy{
					api_v0.Policy{
						Source: api_v0.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: api_v0.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Port:     0,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("missing port"))
			})
		})

		Context("when a tag is supplied", func() {
			It("returns a useful error", func() {
				policies := []api_v0.Policy{
					{
						Source: api_v0.Source{
							ID:  "foo",
							Tag: "some-tag",
						},
						Destination: api_v0.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Port:     42,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("tags may not be specified"))
			})
		})
	})
})
