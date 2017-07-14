package handlers_test

import (
	"policy-server/handlers"
	"policy-server/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validator", func() {
	var validator handlers.Validator

	BeforeEach(func() {
		validator = handlers.Validator{}
	})

	Describe("ValidatePolicies", func() {
		It("does not error for valid policies", func() {
			policies := []models.Policy{
				models.Policy{
					Source: models.Source{
						ID: "some-source-id",
					},
					Destination: models.Destination{
						ID:       "some-destination-id",
						Protocol: "tcp",
						Ports: models.Ports{
							Start: 42,
							End:   123,
						},
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
				err := validator.ValidatePolicies([]models.Policy{})
				Expect(err).To(MatchError("missing policies"))
			})
		})

		Context("when source id is missing", func() {
			It("returns a useful error", func() {
				policies := []models.Policy{
					models.Policy{
						Source: models.Source{
							ID:  "",
							Tag: "",
						},
						Destination: models.Destination{
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
				policies := []models.Policy{
					models.Policy{
						Source: models.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: models.Destination{
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
				policies := []models.Policy{
					models.Policy{
						Source: models.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: models.Destination{
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

		Context("when the end port is less than the start port", func() {
			It("returns a useful error", func() {
				policies := []models.Policy{
					models.Policy{
						Source: models.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: models.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Ports: models.Ports{
								Start: 1243,
								End:   999,
							},
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid port range 1243-999, start must be less than or equal to end"))
			})
		})

		Context("when the start port is less than or equal to 0", func() {
			It("returns a useful error", func() {
				policies := []models.Policy{
					models.Policy{
						Source: models.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: models.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Ports: models.Ports{
								Start: -42,
								End:   999,
							},
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid start port -42, must be in range 1-65535"))
			})
		})

		Context("when the end port is greater than 65535", func() {
			It("returns a useful error", func() {
				policies := []models.Policy{
					models.Policy{
						Source: models.Source{
							ID:  "foo",
							Tag: "",
						},
						Destination: models.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Ports: models.Ports{
								Start: 42,
								End:   101101,
							},
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid end port 101101, must be in range 1-65535"))
			})
		})

		Context("when a tag is supplied", func() {
			It("returns a useful error", func() {
				policies := []models.Policy{
					{
						Source: models.Source{
							ID:  "foo",
							Tag: "some-tag",
						},
						Destination: models.Destination{
							ID:       "bar",
							Tag:      "",
							Protocol: "tcp",
							Ports: models.Ports{
								Start: 123,
								End:   456,
							},
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("tags may not be specified"))
			})
		})
	})
})
