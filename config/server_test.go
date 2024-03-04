package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestServerConfig_Setup(t *testing.T) {
	// Create an instance of ServerConfig
	serverConfig := &ServerConfig{}
	
	serverConfig.Setup()

	// Assert that the App field is not nil
	assert.NotNil(t, serverConfig.App)

}
