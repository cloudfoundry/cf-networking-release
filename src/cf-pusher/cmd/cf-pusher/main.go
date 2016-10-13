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
	"log"
	"os"
	"path/filepath"
)

type ScaleGroup struct {
	Org            string   `json:"org"`
	Space          string   `json:"space"`
	TickApps       []string `json:"tick-apps"`
	TickInstances  int      `json:"tick-instances"`
	Registry       string   `json:"registry"`
	ProxyApp       string   `json:"proxy-app"`
	ProxyInstances int      `json:"proxy-instances"`
}

const prefix = "scale-"

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

	var config config.Config
	if err := json.Unmarshal(configBytes, &config); err != nil {
		log.Fatalf("error unmarshaling config: %s", err)
	}

	var tickApps []string
	for i := 1; i < config.Applications+1; i++ {
		tickApps = append(tickApps, fmt.Sprintf("%s%s-%d", prefix, "tick", i))
	}

	scaleGroup := ScaleGroup{
		Org:            prefix + "org",
		Space:          prefix + "space",
		TickApps:       tickApps,
		TickInstances:  config.AppInstances,
		Registry:       prefix + "registry",
		ProxyApp:       prefix + "proxy",
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
	if err := apiConnector.Connect(); err != nil {
		log.Fatalf("connecting to api: %s", err)
	}

	orgDeleter := &cf_command.OrgDeleter{
		Org:     scaleGroup.Org,
		Adapter: adapter,
	}
	if err = orgDeleter.Delete(); err != nil {
		log.Fatalf("deleting org: %s", err)
	}

	orgSpaceCreator := &cf_command.OrgSpaceCreator{
		Org:     scaleGroup.Org,
		Space:   scaleGroup.Space,
		Adapter: adapter,
	}
	if err = orgSpaceCreator.Create(); err != nil {
		log.Fatalf("creating org and space: %s", err)
	}

	proxyApp := cf_command.Application{
		Name:      "proxy",
		Directory: filepath.Join(appsDir, "proxy"),
	}

	registryApp := cf_command.Application{
		Name:      "registry",
		Directory: filepath.Join(appsDir, "registry"),
	}

	appsToPush := []cf_command.Application{proxyApp, registryApp}

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
				ListenPorts:     3,
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

	manifestGenerator := &manifest_generator.ManifestGenerator{}
	appPusher := cf_command.AppPusher{
		Applications:      appsToPush,
		Adapter:           adapter,
		ManifestGenerator: manifestGenerator,
	}

	if err := appPusher.Push(); err != nil {
		log.Fatalf("pushing apps: %s", err)
	}

	appChecker := cf_command.AppChecker{
		Applications: appsToPush,
		Adapter:      adapter,
	}
	if err := appChecker.CheckApps(); err != nil {
		log.Fatalf("checking apps: %s", err)
	}

	output, err := json.Marshal(scaleGroup)
	if err != nil {
		log.Fatalf("%s", err)
	}
	fmt.Printf("%+v", string(output))
}
