package main_test

import (
	"os/exec"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	runRoutePopulator := func(nats, backendHost string, backendPort int, appDomain, appName string, numRoutes int) *gexec.Session {
		routePopulatorCommand := exec.Command(httpRoutePopulatorPath,
			"-nats", nats,
			"-backendHost", backendHost,
			"-backendPort", strconv.Itoa(backendPort),
			"-appDomain", appDomain,
			"-appName", appName,
			"-numRoutes", strconv.Itoa(numRoutes),
		)
		session, err := gexec.Start(routePopulatorCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		return session
	}

	Context("Argument handling", func() {
		It("errors if no arguments are passed", func() {
			session := runRoutePopulator("", "", 0, "", "", 0)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-nats must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendHost must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendPort must be provided"))
			Expect(session.Err).To(gbytes.Say("-appDomain must be provided"))
			Expect(session.Err).To(gbytes.Say("-appName must be provided"))
			Expect(session.Err).To(gbytes.Say("-numRoutes must be provided"))
			Expect(session.Err).ToNot(gbytes.Say("-publishDelay is an invalid string"))
		})

		It("errors if only nats is passed", func() {
			session := runRoutePopulator("nats", "", 0, "", "", 0)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-backendHost must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendPort must be provided"))
			Expect(session.Err).To(gbytes.Say("-appDomain must be provided"))
			Expect(session.Err).To(gbytes.Say("-appName must be provided"))
			Expect(session.Err).To(gbytes.Say("-numRoutes must be provided"))
		})

		It("errors if only backendHost is passed", func() {
			session := runRoutePopulator("", "backend-host", 0, "", "", 0)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-nats must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendPort must be provided"))
			Expect(session.Err).To(gbytes.Say("-appDomain must be provided"))
			Expect(session.Err).To(gbytes.Say("-appName must be provided"))
			Expect(session.Err).To(gbytes.Say("-numRoutes must be provided"))
		})

		It("errors if only backendPort is passed", func() {
			session := runRoutePopulator("", "", 1234, "", "", 0)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-nats must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendHost must be provided"))
			Expect(session.Err).To(gbytes.Say("-appDomain must be provided"))
			Expect(session.Err).To(gbytes.Say("-appName must be provided"))
			Expect(session.Err).To(gbytes.Say("-numRoutes must be provided"))
		})

		It("errors if only appDomain is passed", func() {
			session := runRoutePopulator("", "", 0, "blah.com", "", 0)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-nats must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendHost must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendPort must be provided"))
			Expect(session.Err).To(gbytes.Say("-appName must be provided"))
			Expect(session.Err).To(gbytes.Say("-numRoutes must be provided"))
		})

		It("errors if only appName is passed", func() {
			session := runRoutePopulator("", "", 0, "", "blah", 0)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-nats must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendHost must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendPort must be provided"))
			Expect(session.Err).To(gbytes.Say("-appDomain must be provided"))
			Expect(session.Err).To(gbytes.Say("-numRoutes must be provided"))
		})

		It("errors if only numRoutes is passed", func() {
			session := runRoutePopulator("", "", 0, "", "", 1)
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-nats must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendHost must be provided"))
			Expect(session.Err).To(gbytes.Say("-backendPort must be provided"))
			Expect(session.Err).To(gbytes.Say("-appDomain must be provided"))
			Expect(session.Err).To(gbytes.Say("-appName must be provided"))
		})
		It("errors if heartbeatInterval is 0", func() {
			routePopulatorCommand := exec.Command(httpRoutePopulatorPath,
				"-heartbeatInterval", "0",
			)
			session, err := gexec.Start(routePopulatorCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-heartbeatInterval must be greater than 0"))
		})

		It("errors if parseDuration is an invalid string", func() {
			routePopulatorCommand := exec.Command(httpRoutePopulatorPath,
				"-publishDelay", "foo",
			)
			session, err := gexec.Start(routePopulatorCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).To(gbytes.Say("-publishDelay is an invalid string"))
		})
	})
})
