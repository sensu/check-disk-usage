package main

import (
	"fmt"
	"math"

	human "github.com/dustin/go-humanize"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
	"github.com/shirou/gopsutil/disk"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	IncludeFSType []string
	ExcludeFSType []string
	IncludeFSPath []string
	ExcludeFSPath []string
	Warning       float64
	Critical      float64
	Normal        float64
	Magic         float64
	Minimum       float64
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-disk-usage",
			Short:    "Cross platform disk usage check for Sensu",
			Keyspace: "sensu.io/plugins/check-disk-usage/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "include-fs-type",
			Env:       "",
			Argument:  "include-fs-type",
			Shorthand: "i",
			Default:   []string{},
			Usage:     "Comma separated list of file system types to check",
			Value:     &plugin.IncludeFSType,
		},
		&sensu.PluginConfigOption{
			Path:      "exclude-fs-type",
			Env:       "",
			Argument:  "exclude-fs-type",
			Shorthand: "e",
			Default:   []string{},
			Usage:     "Comma separated list of file system types to exclude from checking",
			Value:     &plugin.ExcludeFSType,
		},
		&sensu.PluginConfigOption{
			Path:      "include-fs-path",
			Env:       "",
			Argument:  "include-fs-path",
			Shorthand: "I",
			Default:   []string{},
			Usage:     "Comma separated list of file system paths to check",
			Value:     &plugin.IncludeFSPath,
		},
		&sensu.PluginConfigOption{
			Path:      "exclude-fs-path",
			Env:       "",
			Argument:  "exclude-fs-path",
			Shorthand: "E",
			Default:   []string{},
			Usage:     "Comma separated list of file system paths to exclude from checking",
			Value:     &plugin.ExcludeFSPath,
		},
		&sensu.PluginConfigOption{
			Path:      "warning",
			Env:       "",
			Argument:  "warning",
			Shorthand: "w",
			Default:   float64(85),
			Usage:     "Warning threshold for file system usage",
			Value:     &plugin.Warning,
		},
		&sensu.PluginConfigOption{
			Path:      "critical",
			Env:       "",
			Argument:  "critical",
			Shorthand: "c",
			Default:   float64(95),
			Usage:     "Critical threshold for file system usage",
			Value:     &plugin.Critical,
		},
		&sensu.PluginConfigOption{
			Path:      "normal",
			Env:       "",
			Argument:  "normal",
			Shorthand: "n",
			Default:   float64(20),
			Usage:     "Thresholds are not adjusted for filesystems of exactly this size, thresholds are reduced for smaller and increased for larger",
			Value:     &plugin.Normal,
		},
		&sensu.PluginConfigOption{
			Path:      "magic",
			Env:       "",
			Argument:  "magic",
			Shorthand: "m",
			Default:   float64(1.0),
			Usage:     "Magic factor to adjust warning/critical thresholds (example: 0.9, 1.0 == no adjustment)",
			Value:     &plugin.Magic,
		},
		&sensu.PluginConfigOption{
			Path:      "minimum",
			Env:       "",
			Argument:  "minimum",
			Shorthand: "M",
			Default:   float64(100),
			Usage:     "Volumes under this size (in GB) will not be adjusted by magic factor",
			Value:     &plugin.Minimum,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if len(plugin.IncludeFSType) > 0 && len(plugin.ExcludeFSType) > 0 {
		return sensu.CheckStateCritical, fmt.Errorf("--include-fs-type and --exclude-fs-type are mutually exclusive")
	}
	if len(plugin.IncludeFSPath) > 0 && len(plugin.ExcludeFSPath) > 0 {
		return sensu.CheckStateCritical, fmt.Errorf("--include-fs-path and --exclude-fs-path are mutually exclusive")
	}
	if plugin.Warning >= plugin.Critical {
		return sensu.CheckStateCritical, fmt.Errorf("--warning value can not be greater than or equal to --critical value")
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *types.Event) (int, error) {
	var (
		criticals       int
		warnings        int
		criticalPercent float64
		warningPercent  float64
	)

	parts, _ := disk.Partitions(true)

	for _, p := range parts {
		device := p.Mountpoint
		s, _ := disk.Usage(device)

		// Ignore empty file systems
		if s.Total == 0 {
			continue
		}

		// Ignore excluded (or non-included) file system types
		if !isValidFSType(p.Fstype) {
			continue
		}

		// Ignore excluded (or non-included) file systems
		if !isValidFSPath(p.Mountpoint) {
			continue
		}

		// Magic factor adjustment
		if float64(s.Total) < (plugin.Minimum * 1000000000) {
			criticalPercent = plugin.Critical
			warningPercent = plugin.Warning
		} else {
			criticalPercent = adjustPercent(s.Total, plugin.Critical)
			warningPercent = adjustPercent(s.Total, plugin.Warning)
		}

		fmt.Printf("%s ", plugin.PluginConfig.Name)
		if s.UsedPercent >= criticalPercent {
			criticals++
			fmt.Printf("CRITICAL: ")
		} else if s.UsedPercent >= warningPercent {
			warnings++
			fmt.Printf(" WARNING: ")
		} else {
			fmt.Printf("      OK: ")
		}
		fmt.Printf("%s %.2f%% - Total: %s, Used: %s, Free: %s\n", p.Mountpoint, s.UsedPercent, human.Bytes(s.Total), human.Bytes(s.Used), human.Bytes(s.Free))
	}

	if criticals > 0 {
		return sensu.CheckStateCritical, nil
	} else if warnings > 0 {
		return sensu.CheckStateWarning, nil
	}

	return sensu.CheckStateOK, nil
}

func isValidFSType(fsType string) bool {
	// should i account for case insensitve searches for windows (ntfs vs. NTFS)?
	if len(plugin.IncludeFSType) > 0 && contains(plugin.IncludeFSType, fsType) {
		return true
	} else if len(plugin.IncludeFSType) > 0 {
		return false
	} else if len(plugin.ExcludeFSType) > 0 && contains(plugin.ExcludeFSType, fsType) {
		return false
	}

	// either not in exclude list or neither list is specified
	return true
}

func isValidFSPath(fsPath string) bool {
	// should i account for case insensitive searches for windows (c: vs. C:)?
	if len(plugin.IncludeFSPath) > 0 && contains(plugin.IncludeFSPath, fsPath) {
		return true
	} else if len(plugin.IncludeFSPath) > 0 {
		return false
	} else if len(plugin.ExcludeFSPath) > 0 && contains(plugin.ExcludeFSPath, fsPath) {
		return false
	}

	// either not in exclude list or neither list is specified
	return true
}

func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

func adjustPercent(size uint64, percent float64) float64 {
	hsize := (float64(size) / (1024.0 * 1024.0)) / plugin.Normal
	felt := math.Pow(hsize, plugin.Magic)
	scale := felt / hsize
	return 100 - ((100 - percent) * scale)
}
