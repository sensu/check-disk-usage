package main

import (
	"testing"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
}

func TestCheckArgs(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	plugin.IncludeFSType = []string{"ext4", "xfs}"}
	plugin.ExcludeFSType = []string{"tmpfs", "devtmpfs"}
	i, e := checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	plugin.ExcludeFSType = []string{}
	plugin.IncludeFSPath = []string{"/", "/home"}
	plugin.ExcludeFSPath = []string{"/tmp"}
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	plugin.ExcludeFSPath = []string{}
	plugin.Warning = float64(80)
	plugin.Critical = float64(70)
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	plugin.Critical = float64(90)
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateOK, i)
	assert.NoError(e)
}
