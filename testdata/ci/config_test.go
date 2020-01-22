package main

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	KeepFailedApplications bool           `mapstructure:"keep_failed_applications"`
	Alien4Cloud            Alien          `mapstructure:"alien4cloud"`
	Yorc                   Yorc           `mapstructure:"yorc"`
	ChromeDP               ChromeDP       `mapstructure:"chromedp"`
	Infrastructure         Infrastructure `mapstructure:"infrastructure"`
}

type Infrastructure struct {
	Name string `mapstructure:"name"`
}

type Alien struct {
	URL              string `mapstructure:"url"`
	User             string `mapstructure:"user"`
	Password         string `mapstructure:"password"`
	CA               string `mapstructure:"ca"`
	OrchestratorName string `mapstructure:"orchestrator_name"`
}

type Yorc struct {
	ResourcesPrefix string `mapstructure:"resources_prefix"`
}

type ChromeDP struct {
	NoHeadless bool          `mapstructure:"no_headless"`
	Timeout    time.Duration `mapstructure:"timeout"`
}

func initConfig() {
	viper.SetConfigName("godog_config")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	viper.BindEnv("keep_failed_applications", "KEEP_FAILED_APPLICATIONS")
	viper.BindEnv("alien4cloud.url", "ALIEN4CLOUD_URL")
	viper.BindEnv("alien4cloud.user", "ALIEN4CLOUD_USER")
	viper.BindEnv("alien4cloud.password", "ALIEN4CLOUD_PASSWORD")
	viper.BindEnv("alien4cloud.ca", "ALIEN4CLOUD_CA")
	viper.BindEnv("alien4cloud.orchestrator_name", "ALIEN4CLOUD_ORCHESTRATOR_NAME")
	viper.BindEnv("yorc.resources_prefix", "YORC_RESOURCES_PREFIX")
	viper.BindEnv("chromedp.no_headless", "CHROMEDP_NO_HEADLESS")
	viper.BindEnv("chromedp.timeout", "CHROMEDP_TIMEOUT")
	viper.BindEnv("infrastructure.name", "INFRASTRUCTURE_NAME")

	viper.SetDefault("keep_failed_applications", false)
	viper.SetDefault("alien4cloud.url", "http://127.0.0.1:8088")
	viper.SetDefault("alien4cloud.user", "admin")
	viper.SetDefault("alien4cloud.password", "admin")
	viper.SetDefault("alien4cloud.orchestrator_name", "yorc")
	viper.SetDefault("yorc.resources_prefix", "testci-")
	viper.SetDefault("chromedp.no_headless", false)
	viper.SetDefault("chromedp.timeout", time.Minute)
}

func getConfig() (Config, error) {
	c := Config{}

	err := viper.Unmarshal(&c)
	if err != nil {
		return c, fmt.Errorf("failed to unmarshal configuration file: %w", err)
	}

	return c, nil
}
