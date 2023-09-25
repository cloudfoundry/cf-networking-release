package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/cf-pusher/cf_cli_adapter"
	"code.cloudfoundry.org/cf-pusher/cf_command"
	"code.cloudfoundry.org/cf-pusher/config"
	"code.cloudfoundry.org/cf-pusher/manifest_generator"
	"code.cloudfoundry.org/cf-pusher/models"
	"code.cloudfoundry.org/lib/testsupport"
)

type Config struct {
	config.Config
	GlobalAGGs        int  `json:"global_asgs"`
	TotalSpaces       int  `json:"total_spaces"`
	SpacesWithOneASG  int  `json:"spaces_with_one_asg"`
	HowManyASGsIsMany int  `json:"how_many_asgs_is_many"`
	AppsPerSpace      int  `json:"apps_per_space"`
	SkipASGCreation   bool `json:"skip_asg_creation"`
}

type ConcurrentSpaceSetup struct {
	Adapter         *cf_cli_adapter.Adapter
	ApiConnector    cf_command.ApiConnector
	OrgSpaceCreator cf_command.OrgSpaceCreator
	AppPusher       cf_command.AppPusher
}

func main() {
	config := parseConfig()

	globalAdapter := generateAdapterWithHome(config.Prefix)
	globalApiConnector := &cf_command.ApiConnector{
		Api:               config.Api,
		AdminUser:         config.AdminUser,
		AdminPassword:     config.AdminPassword,
		SkipSSLValidation: config.SkipSSLValidation,
		Adapter:           globalAdapter,
	}
	if err := globalApiConnector.Connect(); err != nil {
		log.Fatalf("connecting to api: %s", err)
	}

	var manyASGs []string
	if !config.SkipASGCreation {
		// Create global asgs
		createGlobalASGs(config)
		// Create a bunch of bindable ASGs
		manyASGs = createManyASGs(config.HowManyASGsIsMany, config.ASGSize, config.Prefix, globalAdapter)
	}
	// Compile the proxy app
	compileBinary()

	// Iterate over each space and create/bind asgs and push apps as needed
	sem := make(chan bool, config.Concurrency)
	for i := 0; i < config.TotalSpaces; i++ {
		setup := generateConcurrentSpaceSetup(i, config)
		sem <- true
		go func(s *ConcurrentSpaceSetup, c Config, index int) {
			defer func() { <-sem }()

			// Connect to the api with this adapter
			if err := s.ApiConnector.Connect(); err != nil {
				log.Fatalf("connecting to api: %s", err)
			}

			// Create and target the space
			if err := s.OrgSpaceCreator.Create(); err != nil {
				log.Fatalf("creating org and space: %s", err)
			}

			if index < c.SpacesWithOneASG {
				// Create and bind a single ASG to this space
				createAndBindOneASGToThisSpace(fmt.Sprintf("%s-asg", s.OrgSpaceCreator.Space), c.ASGSize, s.OrgSpaceCreator, s.Adapter)
			} else {
				// Bind many asgs to this space
				bindManyASGsToThisSpace(manyASGs, s.OrgSpaceCreator.Org, s.OrgSpaceCreator.Space, s.Adapter)
			}

			// Push apps for this space
			if err := s.AppPusher.Push(); err != nil {
				log.Printf("Got an error while pushing proxy apps: %s", err)
			}

		}(setup, config, i)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

func generateConcurrentSpaceSetup(spaceNumber int, config Config) *ConcurrentSpaceSetup {
	appsDir := os.Getenv("APPS_DIR")
	if appsDir == "" {
		log.Fatal("APPS_DIR not set")
	}
	orgName := fmt.Sprintf("%s-org", config.Prefix)
	adapter := generateAdapterWithHome(config.Prefix)
	var apps []cf_command.Application
	for i := 0; i < config.AppsPerSpace; i++ {
		apps = append(apps, cf_command.Application{Name: fmt.Sprintf("%s-%s-%d-%d", config.Prefix, "app", spaceNumber, i)})
	}

	return &ConcurrentSpaceSetup{
		Adapter: adapter,
		ApiConnector: cf_command.ApiConnector{
			Api:               config.Api,
			AdminUser:         config.AdminUser,
			AdminPassword:     config.AdminPassword,
			SkipSSLValidation: config.SkipSSLValidation,
			Adapter:           adapter,
		},
		OrgSpaceCreator: cf_command.OrgSpaceCreator{
			Org:   orgName,
			Space: fmt.Sprintf("%s-%s-%d", config.Prefix, "space", spaceNumber),
			Quota: cf_command.Quota{
				Name:             config.Prefix + "-quota",
				Memory:           "1000G",
				InstanceMemory:   -1,
				Routes:           20000,
				ServiceInstances: 100,
				AppInstances:     -1,
				RoutePorts:       -1,
			},
			Adapter: adapter,
		},
		AppPusher: cf_command.AppPusher{
			Applications:            apps,
			Adapter:                 adapter,
			Concurrency:             config.Concurrency,
			ManifestPath:            generateAppManifest(appsDir),
			Directory:               filepath.Join(appsDir, "proxy"),
			SkipIfPresent:           true,
			DesiredRunningInstances: 1,

			PushAttempts:  3,
			RetryWaitTime: 10 * time.Second,
		},
	}
}

func generateAdapterWithHome(prefix string) *cf_cli_adapter.Adapter {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		log.Fatalf("Failed to create a cf home dir")
	}

	return cf_cli_adapter.NewAdapterWithHome(dir)
}

func compileBinary() {
	appsDir := os.Getenv("APPS_DIR")
	if appsDir == "" {
		log.Fatal("APPS_DIR not set")
	}

	buildCmd := exec.Command("go", "build", "-o", "proxy")
	buildCmd.Dir = filepath.Join(appsDir, "proxy")
	buildCmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
	)

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		log.Fatalf("compiling app binary:\nOutput: %s\nError: %s\n", output, err)
	}
}

func generateAppManifest(appsDir string) string {
	manifestGenerator := &manifest_generator.ManifestGenerator{}
	appManifest := models.Manifest{
		Applications: []models.Application{{
			Name:      "proxy",
			Memory:    "32M",
			DiskQuota: "32M",
			BuildPack: "binary_buildpack",
			Instances: 1,
			Command:   "./proxy",
		}},
	}
	manifestPath, err := manifestGenerator.Generate(appManifest)
	if err != nil {
		log.Fatalf("generate manifest: %s", err)
	}

	return manifestPath
}

func createGlobalASGs(config Config) {
	asgContent := testsupport.BuildASG(config.ASGSize)
	asgFile, err := testsupport.CreateTempFile(asgContent)
	if err != nil {
		log.Fatalf("creating asg file: %s", err)
	}

	sem := make(chan bool, config.Concurrency)
	for index := 0; index < config.GlobalAGGs; index++ {
		sem <- true
		go func(p string, i int) {
			defer func() { <-sem }()
			adapter := generateAdapterWithHome(p)
			asgName := fmt.Sprintf("%s-global-%d-asg", p, i)

			// check ASG and install if not OK
			apiConnector := &cf_command.ApiConnector{
				Api:               config.Api,
				AdminUser:         config.AdminUser,
				AdminPassword:     config.AdminPassword,
				SkipSSLValidation: config.SkipSSLValidation,
				Adapter:           adapter,
			}
			if err := apiConnector.Connect(); err != nil {
				log.Fatalf("connecting to api: %s", err)
			}
			asgChecker := cf_command.ASGChecker{Adapter: adapter}
			asgErr := asgChecker.CheckASG(asgName, asgContent)
			if asgErr != nil {
				// install ASG
				asgInstaller := cf_command.ASGInstaller{Adapter: adapter}
				if err = asgInstaller.InstallGlobalASG(asgName, asgFile); err != nil {
					log.Fatalf("install asg: %s", err)
				}
			}
		}(config.Prefix, index)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
}

func bindManyASGsToThisSpace(asgNames []string, orgName, spaceName string, adapter *cf_cli_adapter.Adapter) {
	for _, asg := range asgNames {
		if err := adapter.BindSecurityGroup(asg, orgName, spaceName); err != nil {
			log.Fatalf("binding asg %s to org %s, space %s: %s", asg, orgName, spaceName, err)
		}
	}
}

func createManyASGs(howMany, asgSize int, prefix string, adapter *cf_cli_adapter.Adapter) []string {
	var asgNames []string
	for i := 0; i < howMany; i++ {
		asgName := fmt.Sprintf("%s-many-%d-asg", prefix, i)
		asgNames = append(asgNames, asgName)
		asgContent := testsupport.BuildASG(asgSize)
		asgFile, err := testsupport.CreateTempFile(asgContent)
		if err != nil {
			log.Fatalf("creating asg file: %s", err)
		}

		// check ASG and create if not OK
		asgChecker := cf_command.ASGChecker{Adapter: adapter}
		asgErr := asgChecker.CheckASG(asgName, asgContent)
		if asgErr != nil {
			// install ASG
			if err := adapter.DeleteSecurityGroup(asgName); err != nil {
				log.Fatalf("deleting security group: %s", err)
			}
			if err := adapter.CreateSecurityGroup(asgName, asgFile); err != nil {
				log.Fatalf("creating security group: %s", err)
			}
		}
	}

	return asgNames
}

func createAndBindOneASGToThisSpace(asgName string, asgSize int, osc cf_command.OrgSpaceCreator, adapter *cf_cli_adapter.Adapter) {
	asgContent := testsupport.BuildASG(asgSize)
	asgFile, err := testsupport.CreateTempFile(asgContent)
	if err != nil {
		log.Fatalf("creating asg file: %s", err)
	}

	// check ASG and install if not OK
	asgChecker := cf_command.ASGChecker{Adapter: adapter}
	asgErr := asgChecker.CheckASG(asgName, asgContent)
	if asgErr != nil {
		// install ASG
		asgInstaller := cf_command.ASGInstaller{Adapter: adapter}
		if err = asgInstaller.InstallASG(asgName, asgFile, osc.Org, osc.Space); err != nil {
			log.Fatalf("install asg: %s", err)
		}
	}
}

func parseConfig() Config {
	configPath := flag.String("config", "", "path to the config file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("must include config file with --config")
	}

	configBytes, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("error reading config: %s", err)
	}

	var config Config
	if err := json.Unmarshal(configBytes, &config); err != nil {
		log.Fatalf("error unmarshaling config: %s", err)
	}

	if config.Prefix == "" {
		config.Prefix = "scale-asg"
	}
	config.Prefix = strings.TrimSuffix(config.Prefix, "-")

	if config.SpacesWithOneASG > config.TotalSpaces {
		log.Fatalf("total_spaces must be greater than or equal to spaces_with_one_asg")
	}

	if config.Concurrency < 1 {
		config.Concurrency = 1
	}

	return config
}
