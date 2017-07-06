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

		Context("when invalid destination ports", func() {
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
								Start: 1234,
								End:   2345,
							},
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid destination port range 1234-2345, start and end must be same"))
			})
		})

		Context("when invalid destination port", func() {
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
							Port:     -1,
						},
					},
				}

				err := validator.ValidatePolicies(policies)
				Expect(err).To(MatchError("invalid destination port value -1, must be 1-65535"))
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
