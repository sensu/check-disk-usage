package main

import (
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"github.com/stretchr/testify/assert"
)

func clearOptions() {
	plugin.IncludeFSType = nil
	plugin.ExcludeFSType = nil
	plugin.IncludeFSPath = nil
	plugin.ExcludeFSPath = nil
	plugin.ExtraTags = nil
	plugin.Warning = 85.0
	plugin.Critical = 95.0
	plugin.MinimumGiB = 100.0
	plugin.NormalGiB = 20.0
	plugin.MetricsMode = false
	plugin.Verbose = false
}

func TestAdjPercent(t *testing.T) {
	clearOptions()
	percent := 80.0
	assert := assert.New(t)
	result := adjPercent(0.0, percent)
	assert.Equal(percent, result)
	plugin.NormalGiB = 0.0
	result = adjPercent(100.0, percent)
	assert.Equal(percent, result)
	plugin.NormalGiB = 20.0
	result = adjPercent(100.0, -1.0)
	assert.Equal(0.0, result)
	plugin.Verbose = true
	result = adjPercent(0.8*plugin.NormalGiB*1.074e+9, percent)
	assert.Greater(percent, result)
	assert.LessOrEqual(0.0, result)
	assert.GreaterOrEqual(100.0, result)
	result = adjPercent(10.0*plugin.NormalGiB*1.074e+9, percent)
	assert.Less(percent, result)
	assert.LessOrEqual(0.0, result)
	assert.GreaterOrEqual(100.0, result)

}

func TestIsValidFSType(t *testing.T) {
	clearOptions()
	assert := assert.New(t)
	plugin.IncludeFSType = []string{"ext4", "xfs"}
	result := isValidFSType("ext4")
	assert.True(result)
	result = isValidFSType("xfs")
	assert.True(result)
	result = isValidFSType("bad")
	assert.False(result)
	clearOptions()
	plugin.ExcludeFSType = []string{"ext4", "xfs"}
	result = isValidFSType("bad")
	assert.True(result)
	result = isValidFSType("ext4")
	assert.False(result)
	result = isValidFSType("xfs")
	assert.False(result)

}

func TestIsValidFSPath(t *testing.T) {
	clearOptions()
	assert := assert.New(t)
	result := isValidFSPath("/mount1")
	assert.True(result)
	plugin.IncludeFSPath = []string{"/mount1/", "/mount2"}
	result = isValidFSPath("/mount1/")
	assert.True(result)
	result = isValidFSPath("/mount2")
	assert.True(result)
	result = isValidFSPath("/mount3")
	assert.False(result)
	clearOptions()
	plugin.ExcludeFSPath = []string{"/mount1", "/mount2"}
	result = isValidFSPath("/mount3")
	assert.True(result)
	result = isValidFSPath("/mount1")
	assert.False(result)
	result = isValidFSPath("/mount2")
	assert.False(result)

}

func TestCheckArgs(t *testing.T) {
	clearOptions()
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
	plugin.ExtraTags = []string{"tmpfs", "devtmpfs"}
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.Error(e)
	plugin.ExtraTags = []string{"key1=val1", "key2=val2"}
	i, e = checkArgs(event)
	assert.Equal(sensu.CheckStateOK, i)
	assert.NoError(e)

}

func TestExecuteCheck(t *testing.T) {
	clearOptions()
	plugin.MetricsMode = true
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")
	i, e := executeCheck(event)
	assert.Equal(sensu.CheckStateOK, i)
	assert.NoError(e)
	clearOptions()
	plugin.Warning = -200.0
	plugin.Critical = -100.0
	// Set minimum extremely high to disable magic adjustment
	plugin.MinimumGiB = 100000.0
	i, e = executeCheck(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.NoError(e)
	// Set minimum low and normal high to enable magic adjustment
	plugin.NormalGiB = 100000.0
	plugin.MinimumGiB = 1.0
	plugin.Verbose = true
	i, e = executeCheck(event)
	assert.Equal(sensu.CheckStateCritical, i)
	assert.NoError(e)
}
