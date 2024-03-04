package app

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoad(t *testing.T) {
	configFile := flag.String("config", "../config.yml", "User Config file from user")

	Load(*configFile)

	assert.NotNil(t, Http)

	assert.Equal(t, configFile, &Http.ConfigFile)
}
