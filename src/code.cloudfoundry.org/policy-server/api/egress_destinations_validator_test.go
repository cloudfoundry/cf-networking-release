package api_test

import (
	"code.cloudfoundry.org/policy-server/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EgressDestinationsValidator", func() {
	var validator api.EgressDestinationsValidator

	BeforeEach(func() {
		validator = api.EgressDestinationsValidator{}
	})

	Describe("ValidateEgressDestinations", func() {
		It("does not error for valid egress destinations", func() {
			destinations := []api.EgressDestination{
				{
					Name:        "meow",
					Description: "a cat",
					Rules: []api.EgressDestinationRule{
						{
							Protocol: "tcp",
							Ports:    "8080-8081",
							IPRanges: "192.0.2.1-192.0.2.1",
						},
						{
							Protocol: "udp",
							Ports:    "8080-8081",
							IPRanges: "192.0.2.1-192.0.2.1",
						},
						{
							Protocol: "all",
							Ports:    "8080-8081",
							IPRanges: "192.0.2.1-192.0.2.1",
						},
					},
				},
			}

			err := validator.ValidateEgressDestinations(destinations)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when no destinations are provided", func() {
			It("returns an error", func() {
				err := validator.ValidateEgressDestinations([]api.EgressDestination{})
				Expect(err).To(MatchError("missing destinations"))
			})
		})

		Context("when no rules are provided for a destination", func() {
			It("returns an error", func() {
				destinations := []api.EgressDestination{
					{
						Name:        "meow",
						Description: "a cat",
						Rules:       []api.EgressDestinationRule{},
					},
				}
				err := validator.ValidateEgressDestinations(destinations)
				Expect(err).To(MatchError("missing rules"))
			})
		})

		Context("when the name is invalid", func() {
			Context("when the name is missing", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8081",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("missing destination name"))
				})

			})
		})

		Context("when the portRange is invalid", func() {
			Context("when no ports are provided", func() {

				Context("when the protocol is not icmp", func() {
					It("returns an error", func() {
						destinations := []api.EgressDestination{
							{
								Name:        "meow",
								Description: "a cat",
								Rules: []api.EgressDestinationRule{
									{
										Protocol: "tcp",
										Ports:    "8080-8080",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
									{
										Protocol: "tcp",
										Ports:    "",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
								},
							},
						}

						err := validator.ValidateEgressDestinations(destinations)
						Expect(err).To(MatchError("missing destination ports"))
					})
				})

				Context("when the protocol is icmp", func() {
					It("doesn't return an error", func() {
						destinations := []api.EgressDestination{
							{
								Name:        "meow",
								Description: "a cat",
								Rules: []api.EgressDestinationRule{
									{
										Protocol: "icmp",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
								},
							},
						}

						err := validator.ValidateEgressDestinations(destinations)
						Expect(err).ToNot(HaveOccurred())
					})
				})
			})

			Context("when multiple port ranges are provided", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080,1234-5678",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8000-9000",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("only one port range is currently supported"))
				})
			})

			Context("when the end port is before the start port", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8000-7000",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid port range 8000-7000, start must be less than or equal to end"))
				})
			})

			Context("when a provided port is negative", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "-7000-1000",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid port , could not convert to an integer"))
				})
			})

			Context("when a provided port isn't one", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-johnwayne",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid port johnwayne, could not convert to an integer"))
				})
			})

			Context("when the ports are out of range", func() {
				It("returns an error", func() {

					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8000-999999",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid end port 999999, must be in range 1-65535"))
				})
			})

			Context("when the protocol is icmp", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "icmp",
									Ports:    "8000-9000",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("ports are not supported for icmp protocol"))
				})
			})
		})

		Context("when the protocol is all", func() {
			Context("when no ports are provided", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "all",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}
					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("missing destination ports"))
				})
			})

			Context("when an icmp type is provided", func() {
				It("doesn't return an error", func() {
					icmpType := 13
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "all",
									IPRanges: "192.0.2.1-192.0.2.1",
									Ports:    "8080-8080",
									ICMPType: &icmpType,
								},
							},
						},
					}
					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(Not(HaveOccurred()))
				})
			})

			Context("when an icmp type is provided", func() {
				It("doesn't return an error", func() {
					icmpCode := 13
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "all",
									IPRanges: "192.0.2.1-192.0.2.1",
									Ports:    "8080-8080",
									ICMPCode: &icmpCode,
								},
							},
						},
					}
					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(Not(HaveOccurred()))
				})
			})
		})

		Context("when the protocol is invalid", func() {
			Context("when no protocol is provided", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Ports:    "8080-8081",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("missing destination protocol"))
				})
			})

			Context("when the protocol is not a supported type", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "meow",
									Ports:    "8080-8081",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid destination protocol 'meow', specify either tcp, udp, or icmp"))
				})
			})
		})

		Context("icmp code and type", func() {
			Context("when the protocol is not icmp", func() {
				Context("when the icmp type is provided", func() {
					It("returns an error", func() {
						icmpType := 13
						destinations := []api.EgressDestination{
							{
								Name:        "meow",
								Description: "a cat",
								Rules: []api.EgressDestinationRule{
									{
										Protocol: "tcp",
										Ports:    "8080-8080",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
									{
										Protocol: "tcp",
										Ports:    "8080-8081",
										IPRanges: "192.0.2.1-192.0.2.1",
										ICMPType: &icmpType,
									},
								},
							},
						}

						err := validator.ValidateEgressDestinations(destinations)
						Expect(err).To(MatchError("invalid destination: cannot set icmp_type property for destination with protocol 'tcp'"))
					})
				})

				Context("when the icmp code is provided", func() {
					It("returns an error", func() {
						icmpCode := 13
						destinations := []api.EgressDestination{
							{
								Name:        "meow",
								Description: "a cat",
								Rules: []api.EgressDestinationRule{
									{
										Protocol: "tcp",
										Ports:    "8080-8080",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
									{
										Protocol: "udp",
										Ports:    "8080-8081",
										IPRanges: "192.0.2.1-192.0.2.1",
										ICMPCode: &icmpCode,
									},
								},
							},
						}

						err := validator.ValidateEgressDestinations(destinations)
						Expect(err).To(MatchError("invalid destination: cannot set icmp_code property for destination with protocol 'udp'"))
					})
				})
			})
			Context("when the protocol is icmp", func() {
				Context("when icmp type and code are not provided", func() {
					It("does not error", func() {

						destinations := []api.EgressDestination{
							{
								Name:        "meow",
								Description: "a cat",
								Rules: []api.EgressDestinationRule{
									{
										Protocol: "icmp",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
								},
							},
						}

						err := validator.ValidateEgressDestinations(destinations)
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("when icmp type and code are  provided", func() {
					It("does not error", func() {
						icmpCode := 2
						icmpType := 2

						destinations := []api.EgressDestination{
							{
								Name:        "meow",
								Description: "a cat",
								Rules: []api.EgressDestinationRule{
									{
										Protocol: "tcp",
										Ports:    "8080-8080",
										IPRanges: "192.0.2.1-192.0.2.1",
									},
									{
										Protocol: "icmp",
										IPRanges: "192.0.2.1-192.0.2.1",
										ICMPCode: &icmpCode,
										ICMPType: &icmpType,
									},
								},
							},
						}

						err := validator.ValidateEgressDestinations(destinations)
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
		})

		Context("when the IPRange is invalid", func() {

			Context("when no ips are provided", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8080-8081",
									IPRanges: "",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("missing destination IP range"))
				})
			})

			Context("when the End IP is greater than the Start IP ", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8080-8081",
									IPRanges: "192.0.2.10-192.0.2.11,192.0.2.10-192.0.2.11",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("only one IP range is currently supported"))
				})
			})

			Context("when the IP is not valid ip address", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8080-8081",
									IPRanges: "192.0.2.1-192.0.2.500",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid ip address '192.0.2.500', must be a valid IPv4 address"))
				})
			})

			Context("when the IP is an IPv6 address", func() {
				It("returns an error", func() {

					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8080-8081",
									IPRanges: "2001:0db8:85a3:0000:0000:8a2e:0370:7334-2001:0db8:85a3:0000:0000:8a2e:0370:7334",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid ip address '2001:0db8:85a3:0000:0000:8a2e:0370:7334', must be a valid IPv4 address"))
				})
			})

			Context("when the End IP is greater than the Start IP ", func() {
				It("returns an error", func() {
					destinations := []api.EgressDestination{
						{
							Name:        "meow",
							Description: "a cat",
							Rules: []api.EgressDestinationRule{
								{
									Protocol: "tcp",
									Ports:    "8080-8080",
									IPRanges: "192.0.2.1-192.0.2.1",
								},
								{
									Protocol: "tcp",
									Ports:    "8080-8081",
									IPRanges: "192.0.2.10-192.0.2.1",
								},
							},
						},
					}

					err := validator.ValidateEgressDestinations(destinations)
					Expect(err).To(MatchError("invalid IP range 192.0.2.10-192.0.2.1, start must be less than or equal to end"))
				})
			})
		})
	})
})
