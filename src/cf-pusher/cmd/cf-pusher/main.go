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
	ProxyInstances int      `json:"proxy-instances"` // TODO This doesn't actually do anything, we always assume it's 1
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

	var tickApps []string
	for i := 0; i < config.Applications; i++ {
		tickApps = append(tickApps, fmt.Sprintf("%s%s-%d", prefix, "tick", i))
	}

	var proxyApps []string
	for i := 0; i < config.ProxyApplications; i++ {
		proxyApps = append(proxyApps, fmt.Sprintf("%s%s-%d", prefix, "proxy", i))
	}

	scaleGroup := ScaleGroup{
		Org:            prefix + "org",
		Space:          prefix + "space",
		TickApps:       tickApps,
		TickInstances:  config.AppInstances,
		Registry:       prefix + "registry",
		ProxyApps:      proxyApps,
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
		Memory:           "100G",
		InstanceMemory:   -1,
		Routes:           10000,
		ServiceInstances: 100,
		AppInstances:     -1,
		RoutePorts:       -1,
	}

	registryApp := cf_command.Application{
		Name:      scaleGroup.Registry,
		Directory: filepath.Join(appsDir, "registry"),
	}
	appsToPush := []cf_command.Application{registryApp}

	tickManifest := models.Manifest{
		Applications: []models.Application{{
			Name:      "tick",
			Memory:    "32M",
			DiskQuota: "32M",
			BuildPack: "go_buildpack",
			Instances: scaleGroup.TickInstances,
			Env: models.TickEnvironment{
				GoPackageName:   "example-apps/tick",
				RegistryBaseURL: "http://" + registryApp.Name + "." + config.AppsDomain,
				StartPort:       7000,
				ListenPorts:     config.ExtraListenPorts,
			},
		}},
	}

	for _, tickApp := range scaleGroup.TickApps {
		t := cf_command.Application{
			Name:      tickApp,
			Directory: filepath.Join(appsDir, "tick"),
			Manifest:  tickManifest,
		}
		appsToPush = append(appsToPush, t)
	}

	for _, proxyApp := range scaleGroup.ProxyApps {
		p := cf_command.Application{
			Name:      proxyApp,
			Directory: filepath.Join(appsDir, "proxy"),
		}
		appsToPush = append(appsToPush, p)
	}

	appChecker := cf_command.AppChecker{
		Org:          scaleGroup.Org,
		Applications: appsToPush,
		Adapter:      adapter,
	}

	orgDeleter := &cf_command.OrgDeleter{
		Org:     scaleGroup.Org,
		Quota:   quota,
		Adapter: adapter,
	}

	orgSpaceCreator := &cf_command.OrgSpaceCreator{
		Org:     scaleGroup.Org,
		Space:   scaleGroup.Space,
		Quota:   quota,
		Adapter: adapter,
	}

	manifestGenerator := &manifest_generator.ManifestGenerator{}
	appPusher := cf_command.AppPusher{
		Applications:      appsToPush,
		Adapter:           adapter,
		ManifestGenerator: manifestGenerator,
		Concurrency:       config.Concurrency,
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

	for i := 0; i < config.Applications; i++ {
		expectedApps[fmt.Sprintf("%sproxy-%d", prefix, i)] = config.ProxyInstances
	}

	expectedASG := testsupport.BuildASG(config.ASGSize)
	asgFile, err := testsupport.CreateASGFile(expectedASG)
	if err != nil {
		log.Fatalf("creating asg file: %s", err)
	}

	// check Apps and ASG and exit if both OK
	asgName := fmt.Sprintf("%sasg", prefix)
	appsErr := appChecker.CheckApps(expectedApps)
	asgErr := asgChecker.CheckASG(asgName, expectedASG)
	if appsErr == nil && (asgErr == nil) {
		success(scaleGroup)
		return
	}

	// re-create org and space
	if err = orgDeleter.Delete(); err != nil {
		log.Fatalf("deleting org: %s", err)
	}
	if err = orgSpaceCreator.Create(); err != nil {
		log.Fatalf("creating org and space: %s", err)
	}

	// install ASG
	if err = asgInstaller.InstallASG(asgName, asgFile, scaleGroup.Org, scaleGroup.Space); err != nil {
		log.Fatalf("install asg: %s", err)
	}

	// push apps
	if err := appPusher.Push(); err != nil {
		log.Printf("Got an error while pushing apps: %s", err)
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
