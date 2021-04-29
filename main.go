package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	human "github.com/dustin/go-humanize"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
	"github.com/shirou/gopsutil/disk"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	IncludeFSType   []string
	ExcludeFSType   []string
	IncludeFSPath   []string
	ExcludeFSPath   []string
	Warning         float64
	Critical        float64
	IncludePseudo   bool
	IncludeReadOnly bool
	FailOnError     bool
	HumanReadable   bool
	MetricsMode     bool
	MetricsFormat   string
	ExtraTags       []string
}

var (
	metrics   = []string{}
	tags      = map[string]string{}
	extraTags = map[string]string{}
	plugin    = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-disk-usage",
			Short:    "Cross platform disk usage check for Sensu",
			Keyspace: "sensu.io/plugins/check-disk-usage/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		{
			Path:      "include-fs-type",
			Env:       "",
			Argument:  "include-fs-type",
			Shorthand: "i",
			Default:   []string{},
			Usage:     "Comma separated list of file system types to check",
			Value:     &plugin.IncludeFSType,
		},
		{
			Path:      "exclude-fs-type",
			Env:       "",
			Argument:  "exclude-fs-type",
			Shorthand: "e",
			Default:   []string{},
			Usage:     "Comma separated list of file system types to exclude from checking",
			Value:     &plugin.ExcludeFSType,
		},
		{
			Path:      "include-fs-path",
			Env:       "",
			Argument:  "include-fs-path",
			Shorthand: "I",
			Default:   []string{},
			Usage:     "Comma separated list of file system paths to check",
			Value:     &plugin.IncludeFSPath,
		},
		{
			Path:      "exclude-fs-path",
			Env:       "",
			Argument:  "exclude-fs-path",
			Shorthand: "E",
			Default:   []string{},
			Usage:     "Comma separated list of file system paths to exclude from checking",
			Value:     &plugin.ExcludeFSPath,
		},
		{
			Path:      "warning",
			Env:       "",
			Argument:  "warning",
			Shorthand: "w",
			Default:   float64(85),
			Usage:     "Warning threshold for file system usage",
			Value:     &plugin.Warning,
		},
		{
			Path:      "critical",
			Env:       "",
			Argument:  "critical",
			Shorthand: "c",
			Default:   float64(95),
			Usage:     "Critical threshold for file system usage",
			Value:     &plugin.Critical,
		},
		{
			Path:      "include-pseudo-fs",
			Env:       "",
			Argument:  "include-pseudo-fs",
			Shorthand: "p",
			Default:   false,
			Usage:     "Include pseudo-filesystems (e.g. tmpfs) (default false)",
			Value:     &plugin.IncludePseudo,
		},
		{
			Path:      "fail-on-error",
			Env:       "",
			Argument:  "fail-on-error",
			Shorthand: "f",
			Default:   false,
			Usage:     "Fail and exit on errors getting file system usage (e.g. permission denied) (default false)",
			Value:     &plugin.FailOnError,
		},
		{
			Path:      "include-read-only",
			Env:       "",
			Argument:  "include-read-only",
			Shorthand: "r",
			Default:   false,
			Usage:     "Include read-only filesystems (default false)",
			Value:     &plugin.IncludeReadOnly,
		},
		{
			Path:      "human-readable",
			Env:       "",
			Argument:  "human-readable",
			Shorthand: "H",
			Default:   false,
			Usage:     "print sizes in powers of 1024 (default false)",
			Value:     &plugin.HumanReadable,
		},
		{
			Path:     "metrics",
			Env:      "",
			Argument: "metrics",
			Default:  false,
			Usage:    "Output metrics instead of human readable output",
			Value:    &plugin.MetricsMode,
		},
		{
			Path:     "metrics-format",
			Env:      "",
			Argument: "metrics-format",
			Default:  "opentsdb_line",
			Usage:    "Metrics output format, supports opentsdb_line or prometheus_text",
			Value:    &plugin.MetricsFormat,
		},
		{
			Path:     "tags",
			Env:      "",
			Argument: "tags",
			Default:  []string{},
			Usage:    "Comma separated list of additional metrics tags using key=value format.",
			Value:    &plugin.ExtraTags,
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
	for _, tagString := range plugin.ExtraTags {
		fmt.Println(tagString)
		parts := strings.Split(tagString, `=`)
		if len(parts) == 2 {
			extraTags[parts[0]] = parts[1]
		} else {
			return sensu.CheckStateCritical, fmt.Errorf("Failed to parse input tag: %s", tagString)
		}
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *types.Event) (int, error) {
	var (
		criticals int
		warnings  int
	)
	timeNow := time.Now().Unix()
	parts, err := disk.Partitions(plugin.IncludePseudo)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Failed to get partitions, error: %v", err)
	}

	for _, p := range parts {
		tags = map[string]string{}
		for key, value := range extraTags {
			tags[key] = value
		}
		// Ignore excluded (or non-included) file system types
		if !isValidFSType(p.Fstype) {
			continue
		}

		// Ignore excluded (or non-included) file systems
		if !isValidFSPath(p.Mountpoint) {
			continue
		}

		// Ignore read-only file systems?
		if !plugin.IncludeReadOnly && isReadOnly(p.Opts) {
			continue
		}

		tags["mountpoint"] = p.Mountpoint
		device := p.Mountpoint
		s, err := disk.Usage(device)
		if err != nil {
			if plugin.FailOnError {
				return sensu.CheckStateCritical, fmt.Errorf("Failed to get disk usage for %s, error: %v", device, err)
			}
			if !plugin.MetricsMode {
				fmt.Printf("%s  UNKNOWN: %s - error: %v\n", plugin.PluginConfig.Name, device, err)
			}
			continue
		}

		// Ignore empty file systems
		if s.Total == 0 {
			continue
		}

		// implement magic factor for larger file systems?
		if !plugin.MetricsMode {
			fmt.Printf("%s ", plugin.PluginConfig.Name)
		}
		if s.UsedPercent >= plugin.Critical {
			criticals++
			addMetric("disk.critical", tags, fmt.Sprintf("%v", 1), timeNow)
			addMetric("disk.warning", tags, fmt.Sprintf("%v", 0), timeNow)
			if !plugin.MetricsMode {
				fmt.Printf("CRITICAL: ")
			}
		} else if s.UsedPercent >= plugin.Warning {
			warnings++
			addMetric("disk.critical", tags, fmt.Sprintf("%v", 0), timeNow)
			addMetric("disk.warning", tags, fmt.Sprintf("%v", 1), timeNow)
			if !plugin.MetricsMode {
				fmt.Printf(" WARNING: ")
			}
		} else {
			addMetric("disk.critical", tags, fmt.Sprintf("%v", 0), timeNow)
			addMetric("disk.warning", tags, fmt.Sprintf("%v", 0), timeNow)
			if !plugin.MetricsMode {
				fmt.Printf("      OK: ")
			}
		}

		if !plugin.MetricsMode {
			if plugin.HumanReadable {
				fmt.Printf("%s %.2f%% - Total: %s, Used: %s, Free: %s\n",
					p.Mountpoint, s.UsedPercent, human.IBytes(s.Total), human.IBytes(s.Used), human.IBytes(s.Free))
			} else {
				fmt.Printf("%s %.2f%% - Total: %s, Used: %s, Free: %s\n",
					p.Mountpoint, s.UsedPercent, human.Bytes(s.Total), human.Bytes(s.Used), human.Bytes(s.Free))
			}
		}
		addMetric("disk.percent_used", tags, fmt.Sprintf("%.3f", s.UsedPercent), timeNow)
		addMetric("disk.total_bytes", tags, fmt.Sprintf("%v", s.Total), timeNow)
		addMetric("disk.used_bytes", tags, fmt.Sprintf("%v", s.Used), timeNow)
		addMetric("disk.free_bytes", tags, fmt.Sprintf("%v", s.Free), timeNow)
	}
	tags = map[string]string{}
	for key, value := range extraTags {
		tags[key] = value
	}
	tags["mountpoint"] = "all"
	addMetric("disk.critical", tags, fmt.Sprintf("%v", criticals), timeNow)
	addMetric("disk.warning", tags, fmt.Sprintf("%v", warnings), timeNow)
	if plugin.MetricsMode {
		for _, metric := range metrics {
			fmt.Println(metric)
		}

	}
	if criticals > 0 {
		return sensu.CheckStateCritical, nil
	} else if warnings > 0 {
		return sensu.CheckStateWarning, nil
	}

	return sensu.CheckStateOK, nil
}

func isValidFSType(fsType string) bool {
	// should i account for case insensitive searches for windows (ntfs vs. NTFS)?
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

func isReadOnly(mountOpts string) bool {
	mOpts := strings.Split(mountOpts, ",")
	// "ro" should cover Linux, macOS, and Windows, "read-only" is reportd by mount(8)
	// on macOS so check for it, just in case
	if contains(mOpts, "ro") || contains(mOpts, "read-only") {
		return true
	}
	return false
}

func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

func addMetric(metricName string, tags map[string]string, value string, timeNow int64) {
	switch plugin.MetricsFormat {
	case "opentsdb_line":
		addOpenTSDBMetric(metricName, tags, value, timeNow)
	case "prometheus_text":
		addPrometheusMetric(metricName, tags, value, timeNow)
	default:
		addOpenTSDBMetric(metricName, tags, value, timeNow)
	}
}

func addOpenTSDBMetric(metricName string, tags map[string]string, value string, timeNow int64) {
	tagStr := ""
	for tag, tvalue := range tags {
		if len(tagStr) > 0 {
			tagStr = tagStr + " "
		}
		tagStr = tagStr + tag + "=" + tvalue
	}
	outputs := []string{metricName, strconv.FormatInt(timeNow, 10), value, tagStr}
	//fmt.Println(strings.Join(outputs, " "))
	metrics = append(metrics, strings.Join(outputs, " "))
}

func addPrometheusMetric(metricName string, tags map[string]string, value string, timeNow int64) {
	tagStr := ""
	for tag, tvalue := range tags {
		if len(tagStr) > 0 {
			tagStr = tagStr + ","
		}
		tagStr = tagStr + tag + "=\"" + tvalue + "\""
	}
	if len(tagStr) > 0 {
		tagStr = "{" + tagStr + "}"
	}
	metricName = strings.Replace(metricName, ".", "_", -1)
	outputs := []string{metricName + tagStr, value, strconv.FormatInt(timeNow, 10)}
	//fmt.Println(strings.Join(outputs, " "))
	metrics = append(metrics, strings.Join(outputs, " "))
}
