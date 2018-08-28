package main

import (
	"cf-pusher/cf_cli_adapter"
	"cf-pusher/cf_command"
	"cf-pusher/config"
	"cf-pusher/manifest_generator"
	"cf-pusher/models"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/testsupport"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ScaleGroup struct {
	Org            string   `json:"org"`
	Space          string   `json:"space"`
	TickApps       []string `json:"tick-apps"`
	TickInstances  int      `json:"tick-instances"`
	Registry       string   `json:"registry"`
	ProxyApps      []string `json:"proxy-apps"`
	ProxyInstances int      `json:"proxy-instances"`
}

func main() {
	appsDir := os.Getenv("APPS_DIR")
	if appsDir == "" {
		log.Fatal("APPS_DIR not set")
	}

	configPath := flag.String("config", "", "path to the config file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("must include config file with --config")
	}

	configBytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("error reading config: %s", err)
	}

	config := config.Config{
		Concurrency: 16,
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		log.Fatalf("error unmarshaling config: %s", err)
	}

	var prefix string
	if config.Prefix == "" {
		prefix = "scale-"
	} else {
		prefix = config.Prefix
	}

	var tickAppNames []string
	for i := 0; i < config.Applications; i++ {
		tickAppNames = append(tickAppNames, fmt.Sprintf("%s%s-%d", prefix, "tick", i))
	}

	var proxyAppNames []string
	for i := 0; i < config.ProxyApplications; i++ {
		proxyAppNames = append(proxyAppNames, fmt.Sprintf("%s%s-%d", prefix, "proxy", i))
	}

	scaleGroup := ScaleGroup{
		Org:            prefix + "org",
		Space:          prefix + "space",
		TickApps:       tickAppNames,
		TickInstances:  config.AppInstances,
		Registry:       prefix + "registry",
		ProxyApps:      proxyAppNames,
		ProxyInstances: config.ProxyInstances,
	}

	adapter := &cf_cli_adapter.Adapter{
		CfCliPath: "cf",
	}
	apiConnector := &cf_command.ApiConnector{
		Api:               config.Api,
		AdminUser:         config.AdminUser,
		AdminPassword:     config.AdminPassword,
		SkipSSLValidation: config.SkipSSLValidation,
		Adapter:           adapter,
	}

	quota := cf_command.Quota{
		Name:             prefix + "quota",
		Memory:           "1000G",
		InstanceMemory:   -1,
		Routes:           20000,
		ServiceInstances: 100,
		AppInstances:     -1,
		RoutePorts:       -1,
	}

	manifestGenerator := &manifest_generator.ManifestGenerator{}

	registryAppDirectory := filepath.Join(appsDir, "registry")
	registryManifestPath := filepath.Join(registryAppDirectory, "manifest.yml")
	if err != nil {
		log.Fatal("generate manifest: %s", err)
	}
	registryApp := cf_command.Application{
		Name: scaleGroup.Registry,
	}

	tickAppManifest := models.Manifest{
		Applications: []models.Application{{
			Name:      "tick",
			Memory:    "32M",
			DiskQuota: "32M",
			BuildPack: "go_buildpack",
			Instances: scaleGroup.TickInstances,
			Env: models.TickEnvironment{
				GoPackageName:      "example-apps/tick",
				RegistryBaseURL:    "http://" + registryApp.Name + "." + config.AppsDomain,
				RegistryTTLSeconds: config.AppRegistryTTLSeconds,
				StartPort:          7000,
				ListenPorts:        config.ExtraListenPorts,
			},
		}},
	}
	tickApps := []cf_command.Application{}
	tickAppDirectory := filepath.Join(appsDir, "tick")
	tickManifestPath, err := manifestGenerator.Generate(tickAppManifest)
	if err != nil {
		log.Fatal("generate manifest: %s", err)
	}
	for _, tickApp := range scaleGroup.TickApps {
		t := cf_command.Application{
			Name: tickApp,
		}
		tickApps = append(tickApps, t)
	}

	proxyAppManifest := models.Manifest{
		Applications: []models.Application{{
			Name:      "proxy",
			Memory:    "32M",
			DiskQuota: "32M",
			BuildPack: "go_buildpack",
			Instances: scaleGroup.ProxyInstances,
			Env: models.ProxyEnvironment{
				GoPackageName: "example-apps/proxy",
			},
		}},
	}
	proxyApps := []cf_command.Application{}
	proxyAppDirectory := filepath.Join(appsDir, "proxy")
	proxyManifestPath, err := manifestGenerator.Generate(proxyAppManifest)
	if err != nil {
		log.Fatal("generate manifest: %s", err)
	}
	for _, proxyApp := range scaleGroup.ProxyApps {
		p := cf_command.Application{
			Name: proxyApp,
		}
		proxyApps = append(proxyApps, p)
	}

	appChecker := cf_command.AppChecker{
		Org:          scaleGroup.Org,
		Applications: append(append(proxyApps, registryApp), tickApps...),
		Adapter:      adapter,
		Concurrency:  config.Concurrency,
	}

	orgChecker := &cf_command.OrgChecker{
		Org:     scaleGroup.Org,
		Adapter: adapter,
	}

	orgSpaceCreator := &cf_command.OrgSpaceCreator{
		Org:     scaleGroup.Org,
		Space:   scaleGroup.Space,
		Quota:   quota,
		Adapter: adapter,
	}

	registryAppPusher := cf_command.AppPusher{
		Applications:            []cf_command.Application{registryApp},
		Adapter:                 adapter,
		Concurrency:             config.Concurrency,
		ManifestPath:            registryManifestPath,
		Directory:               registryAppDirectory,
		SkipIfPresent:           true,
		DesiredRunningInstances: 1,
	}
	tickAppPusher := cf_command.AppPusher{
		Applications:            tickApps,
		Adapter:                 adapter,
		Concurrency:             config.Concurrency,
		ManifestPath:            tickManifestPath,
		Directory:               tickAppDirectory,
		SkipIfPresent:           true,
		DesiredRunningInstances: scaleGroup.TickInstances,
	}
	proxyAppPusher := cf_command.AppPusher{
		Applications:            proxyApps,
		Adapter:                 adapter,
		Concurrency:             config.Concurrency,
		ManifestPath:            proxyManifestPath,
		Directory:               proxyAppDirectory,
		SkipIfPresent:           true,
		DesiredRunningInstances: scaleGroup.ProxyInstances,
	}

	asgChecker := cf_command.ASGChecker{
		Adapter: adapter,
	}

	asgInstaller := cf_command.ASGInstaller{
		Adapter: adapter,
	}

	// connect to org and space
	if err := apiConnector.Connect(); err != nil {
		log.Fatalf("connecting to api: %s", err)
	}
	adapter.TargetOrg(scaleGroup.Org)
	adapter.TargetSpace(scaleGroup.Space)

	// declare what apps we expect
	expectedApps := map[string]int{
		prefix + "registry": 1,
	}

	for i := 0; i < config.Applications; i++ {
		expectedApps[fmt.Sprintf("%stick-%d", prefix, i)] = config.AppInstances
	}

	for i := 0; i < config.ProxyApplications; i++ {
		expectedApps[fmt.Sprintf("%sproxy-%d", prefix, i)] = config.ProxyInstances
	}

	expectedASG := testsupport.BuildASG(config.ASGSize)
	asgFile, err := testsupport.CreateTempFile(expectedASG)
	if err != nil {
		log.Fatalf("creating asg file: %s", err)
	}

	if !orgChecker.CheckOrgExists() {
		if err = orgSpaceCreator.Create(); err != nil {
			log.Fatalf("creating org and space: %s", err)
		}
	}

	// check ASG and install if not OK
	asgName := fmt.Sprintf("%sasg", prefix)
	asgErr := asgChecker.CheckASG(asgName, expectedASG)
	if asgErr != nil {
		// install ASG
		if err = asgInstaller.InstallASG(asgName, asgFile, scaleGroup.Org, scaleGroup.Space); err != nil {
			log.Fatalf("install asg: %s", err)
		}
	}

	// push apps
	if err := registryAppPusher.Push(); err != nil {
		log.Printf("Got an error while pushing registry: %s", err)
	}
	if err := tickAppPusher.Push(); err != nil {
		log.Printf("Got an error while pushing tick apps: %s", err)
	}
	if err := proxyAppPusher.Push(); err != nil {
		log.Printf("Got an error while pushing proxy apps: %s", err)
	}

	// check that apps pushed OK
	maxRetries := 5
	for i := 0; ; i++ {
		if err := appChecker.CheckApps(expectedApps); err != nil {
			log.Printf("checking apps: %s\n", err)
			if i == maxRetries {
				log.Fatal("max retries exceeded")
			} else {
				log.Println("checking again in 30 seconds...")
				time.Sleep(30 * time.Second)
			}
		} else {
			break
		}
	}

	success(scaleGroup)
}

func success(scaleGroup ScaleGroup) {
	output, err := json.Marshal(scaleGroup)
	if err != nil {
		log.Fatalf("%s", err)
	}
	fmt.Printf("%+v", string(output))
}
