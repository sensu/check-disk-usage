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
	ExtraTags       []string
}

type MetricGroup struct {
	Comment string
	Type    string
	Name    string
	Metrics []Metric
}

func (g *MetricGroup) AddMetric(tags map[string]string, value float64, timeNow int64) {
	g.Metrics = append(g.Metrics, Metric{
		Tags:      tags,
		Timestamp: timeNow,
		Value:     value,
	})
}
func (g *MetricGroup) Output() {
	var output string
	metricName := strings.Replace(g.Name, ".", "_", -1)
	fmt.Printf("# HELP %s [%s] %s\n", metricName, g.Type, g.Comment)
	fmt.Printf("# TYPE %s %s\n", metricName, g.Type)
	for _, m := range g.Metrics {
		tagStr := ""
		for tag, tvalue := range m.Tags {
			if len(tagStr) > 0 {
				tagStr = tagStr + ","
			}
			tagStr = tagStr + tag + "=\"" + tvalue + "\""
		}
		if len(tagStr) > 0 {
			tagStr = "{" + tagStr + "}"
		}
		output = strings.Join(
			[]string{metricName + tagStr, fmt.Sprintf("%v", m.Value), strconv.FormatInt(m.Timestamp, 10)}, " ")
		fmt.Println(output)
	}
	fmt.Println("")
}

type Metric struct {
	Tags      map[string]string
	Timestamp int64
	Value     float64
}

var (
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

	timeNow := time.Now().UnixNano() / 1000000
	parts, err := disk.Partitions(plugin.IncludePseudo)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Failed to get partitions, error: %v", err)
	}

	metricGroups := map[string]*MetricGroup{
		"disk.critical": &MetricGroup{
			Name:    "disk.critical",
			Type:    "GAUGE",
			Comment: "non-zero value indicates mountpoint usage is above critical threshold",
			Metrics: []Metric{},
		},
		"disk.warning": &MetricGroup{
			Name:    "disk.warning",
			Type:    "GAUGE",
			Comment: "non-zero value indicates mountpoint usage is above warning threshold",
			Metrics: []Metric{},
		},
		"disk.percent_used": &MetricGroup{
			Name:    "disk.percent_usage",
			Type:    "GAUGE",
			Comment: "Percentage of mounted volume used",
			Metrics: []Metric{},
		},
		"disk.total_bytes": &MetricGroup{
			Name:    "disk.total_bytes",
			Type:    "GAUGE",
			Comment: "Total size in bytes of mounted volumed",
			Metrics: []Metric{},
		},
		"disk.used_bytes": &MetricGroup{
			Name:    "disk.used_bytes",
			Type:    "GAUGE",
			Comment: "Used size in bytes of mounted volumed",
			Metrics: []Metric{},
		},
		"disk.free_bytes": &MetricGroup{
			Name:    "disk.free_bytes",
			Type:    "GAUGE",
			Comment: "Free size in bytes of mounted volumed",
			Metrics: []Metric{},
		},
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
		crit := 0
		warn := 0
		if s.UsedPercent >= plugin.Critical {
			criticals++
			crit = 1
		}
		if s.UsedPercent >= plugin.Warning {
			warnings++
			warn = 1
		}
		metricGroups["disk.critical"].AddMetric(tags, float64(crit), timeNow)
		metricGroups["disk.warning"].AddMetric(tags, float64(warn), timeNow)
		if !plugin.MetricsMode {
			fmt.Printf("%s ", plugin.PluginConfig.Name)
			if crit > 0 {
				fmt.Printf("CRITICAL: ")
			} else if warn > 0 {
				fmt.Printf(" WARNING: ")
			} else {
				fmt.Printf("      OK: ")
			}
			if plugin.HumanReadable {
				fmt.Printf("%s %.2f%% - Total: %s, Used: %s, Free: %s\n",
					p.Mountpoint, s.UsedPercent, human.IBytes(s.Total), human.IBytes(s.Used), human.IBytes(s.Free))
			} else {
				fmt.Printf("%s %.2f%% - Total: %s, Used: %s, Free: %s\n",
					p.Mountpoint, s.UsedPercent, human.Bytes(s.Total), human.Bytes(s.Used), human.Bytes(s.Free))
			}
		}
		metricGroups["disk.percent_used"].AddMetric(tags, float64(s.UsedPercent), timeNow)
		metricGroups["disk.total_bytes"].AddMetric(tags, float64(s.Total), timeNow)
		metricGroups["disk.used_bytes"].AddMetric(tags, float64(s.Used), timeNow)
		metricGroups["disk.free_bytes"].AddMetric(tags, float64(s.Free), timeNow)
	}
	tags = map[string]string{}
	for key, value := range extraTags {
		tags[key] = value
	}
	tags["mountpoint"] = "any"
	anyCritical := func() float64 {
		if criticals > 0 {
			return 1
		} else {
			return 0
		}
	}()
	metricGroups["disk.critical"].AddMetric(tags, anyCritical, timeNow)
	anyWarning := func() float64 {
		if warnings > 0 {
			return 1
		} else {
			return 0
		}
	}()
	metricGroups["disk.warning"].AddMetric(tags, anyWarning, timeNow)
	if plugin.MetricsMode {
		// output metrics in a fixed order
		metricGroups["disk.critical"].Output()
		metricGroups["disk.warning"].Output()
		metricGroups["disk.percent_used"].Output()
		metricGroups["disk.total_bytes"].Output()
		metricGroups["disk.used_bytes"].Output()
		metricGroups["disk.free_bytes"].Output()
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
