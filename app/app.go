package app

import (
	"ajbell.co.uk/config"
)

var Http *config.AppConfig

func Load(configFile string) {
	Http = &config.AppConfig{ConfigFile: configFile}

	Http.Setup()

}
