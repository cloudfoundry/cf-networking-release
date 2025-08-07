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
	GlobalASGs                          int  `json:"global_asgs"`
	TotalASGs                           int  `json:"total_asgs"`
	ASGsPerSpace                        int  `json:"spaces_with_one_asg"`
	ASGsWithMultipleSpaces              int  `json:"asgs_with_multiple_spaces"`
	SpaceCountForASGsWithMultipleSpaces int  `json:"space_count_for_asgs_with_multiple_spaces"`
	TotalSpaces                         int  `json:"total_spaces"`
	AppsPerSpace                        int  `json:"apps_per_space"`
	SkipASGCreation                     bool `json:"skip_asg_creation"`
	AppInstancesPerApp                  int  `json:"app_instances_per_app"`
	MaxAppInstances                     int  `json:"max_app_instances"`
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

	// Iterate over each space and create/bind asgs and push apps as needed

	var spaces []string
	var asgs []string
	if !config.SkipASGCreation {
		// Create global asgs
		createGlobalASGs(config)
		// Create a bunch of bindable ASGs
		asgs = createASGs(config.TotalASGs-config.GlobalASGs, config.ASGSize, config.Prefix, globalAdapter)
	}
	spaces = createSpacesConcurrently(config)

	orgName := fmt.Sprintf("%s-org", config.Prefix)

	var spaceIndex = 0
	if !config.SkipASGCreation {
		for _, asg := range asgs[0:config.ASGsWithMultipleSpaces] {
			for range config.SpaceCountForASGsWithMultipleSpaces {
				if spaceIndex >= len(spaces) {
					spaceIndex = 0
				}
				bindASGToThisSpace(asg, orgName, spaces[spaceIndex], globalAdapter)
				spaceIndex++
			}
		}

		for _, asg := range asgs[config.ASGsWithMultipleSpaces:] {
			if spaceIndex >= len(spaces) {
				spaceIndex = 0
			}
			bindASGToThisSpace(asg, orgName, spaces[spaceIndex], globalAdapter)
			spaceIndex++
		}
	}
}

func createSpacesConcurrently(config Config) []string {
	sem := make(chan bool, config.Concurrency)
	var spaceNames []string
	aiCount := 0
	for i := 0; i < config.TotalSpaces; i++ {
		sem <- true
		setup := generateConcurrentSpaceSetup(i, config)
		spaceNames = append(spaceNames, setup.OrgSpaceCreator.Space)
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

			// Push apps for this space
			if aiCount < config.MaxAppInstances {
				if err := s.AppPusher.Push(); err != nil {
					log.Printf("Got an error while pushing proxy apps: %s", err)
				}
				aiCount += config.AppInstancesPerApp
			}
		}(setup, config, i)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}
	return spaceNames
}

func generateConcurrentSpaceSetup(spaceNumber int, config Config) *ConcurrentSpaceSetup {
	appsDir := os.Getenv("APPS_DIR")
	appFolder := os.Getenv("APP_FOLDER")
	if appsDir == "" {
		log.Fatal("APPS_DIR not set")
	}
	if appFolder == "" {
		appFolder = "proxy"
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
			Directory:               filepath.Join(appsDir, appFolder),
			SkipIfPresent:           true,
			DesiredRunningInstances: config.AppInstancesPerApp,

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
			Name:      "cf-nc-app",
			Memory:    "8M",
			DiskQuota: "8M",
			BuildPack: "binary_buildpack",
			Instances: 1,
		}},
	}
	manifestPath, err := manifestGenerator.Generate(appManifest)
	if err != nil {
		log.Fatalf("generate manifest: %s", err)
	}

	return manifestPath
}

func createGlobalASGs(config Config) {
	sem := make(chan bool, config.Concurrency)
	for index := range config.GlobalASGs {
		sem <- true
		go func(p string, i int) {
			defer func() { <-sem }()
			adapter := generateAdapterWithHome(p)
			asgChecker := cf_command.ASGChecker{Adapter: adapter}
			asgInstaller := cf_command.ASGInstaller{Adapter: adapter}
			asgName := fmt.Sprintf("%s-global-%d-asg", p, i)
			asgContent := testsupport.BuildASG(config.ASGSize)
			asgFile, err := testsupport.CreateTempFile(asgContent)
			if err != nil {
				log.Fatalf("creating asg file: %s", err)
			}

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
			asgErr := asgChecker.CheckASG(asgName, asgContent, true, false)
			if asgErr != nil {
				// install ASG
				if err = asgInstaller.InstallGlobalASG(asgName, asgFile); err != nil {
					log.Fatalf("install asg: %s", err)
				}
			}
		}(config.Prefix, index)
	}

	for range cap(sem) {
		sem <- true
	}
}

func bindASGToThisSpace(asg string, orgName, spaceName string, adapter *cf_cli_adapter.Adapter) {
	if err := adapter.BindSecurityGroup(asg, orgName, spaceName); err != nil {
		log.Fatalf("binding asg %s to org %s, space %s: %s", asg, orgName, spaceName, err)
	}
}

func createASGs(howMany, asgSize int, prefix string, adapter *cf_cli_adapter.Adapter) []string {
	var asgNames []string
	for i := range howMany {
		asgName := fmt.Sprintf("%s-many-%d-asg", prefix, i)
		asgNames = append(asgNames, asgName)
		asgContent := testsupport.BuildASG(asgSize)
		asgFile, err := testsupport.CreateTempFile(asgContent)
		if err != nil {
			log.Fatalf("creating asg file: %s", err)
		}

		// check ASG and create if not OK
		asgChecker := cf_command.ASGChecker{Adapter: adapter}
		asgErr := asgChecker.CheckASG(asgName, asgContent, false, false)
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

	if config.SpaceCountForASGsWithMultipleSpaces > config.TotalSpaces {
		log.Fatalf("total_spaces must be greater than or equal to spaces_with_one_asg")
	}

	if config.GlobalASGs+config.ASGsWithMultipleSpaces > config.TotalASGs {
		log.Fatalf("total_spaces must be greater than or equal to spaces_with_one_asg")
	}

	if config.Concurrency < 1 {
		config.Concurrency = 1
	}

	return config
}
