[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/check-disk-usage)
![Go Test](https://github.com/sensu/check-disk-usage/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/check-disk-usage/workflows/goreleaser/badge.svg)

# Sensu disk usage check

## Table of Contents
- [Overview](#overview)
- [Usage](#usage)
  - [Help output](#help-output)
  - [Usage notes](#usage-notes)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Contributing](#contributing)

## Overview

The Sensu disk usage check is a [Sensu Check][2] that reports on disk usage
allowing for the inclusion or exclusion of certain file systems and/or file
system types.

## Usage

### Help output
```
Cross platform disk usage check for Sensu

Usage:
  check-disk-usage [flags]
  check-disk-usage [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -i, --include-fs-type strings   Comma separated list of file system types to check
  -e, --exclude-fs-type strings   Comma separated list of file system types to exclude from checking
  -I, --include-fs-path strings   Comma separated list of file system paths to check
  -E, --exclude-fs-path strings   Comma separated list of file system paths to exclude from checking
  -w, --warning float             Warning threshold for file system usage (default 85)
  -c, --critical float            Critical threshold for file system usage (default 95)
  -p, --include-pseudo-fs         Include pseudo-filesystems (e.g. tmpfs) (default false)
  -r, --include-read-only         Include read-only filesystems (default false)
  -f, --fail-on-error             Fail and exit on errors getting file system usage (e.g. permission denied) (default false)
  -H, --human-readable            print sizes in powers of 1024 (default false)
      --metrics                   Output metrics instead of human readable output
      --tags strings              Comma separated list of additional metrics tags using key=value format.
  -h, --help                      help for check-disk-usage

Use "check-disk-usage [command] --help" for more information about a command.
```

### Usage notes

* The include and exclude options for both file system type and path are
mutually exclusive (e.g. you can not use `--exclude-fs-type` and
`--include-fs-type` on the same check).
* The file system path on Linux/Unix/macOS systems means the file system mount
point (e.g. /, /tmp, /home)
* The file system path on Windows refers to the drive letter (e.g. C:, D:).
Volumes mounted via UNC paths are not checked.
* File system types and paths on Windows are capitalized and need to be
specified as such (e.g. NTFS, C:)
* The `--include-pseudo-fs` option is false by default meaning that on Linux
systems file system with types such as tmpfs (e.g. /dev, /run, etc.) will
be ignored. This takes precedence over any explicit includes or excludes.
* The `--include-read-only` checks for the `ro` mount option on Linux and the
`read-only` mount option on macOS and read-only volumes on Windows.  By default
these file systems will be ignored. The rationale being that if a monitored
system does not have write access to a file system, it cannot be used to create
files (not the source of the problem) nor can it be used to clean up the file
system (not a part of the solution).
* The `--fail-on-error` option determines what occurs if the check encounters an
error, such as `permission denied` for a file system.  If true, the check will
exit with as a critical failure and provide the error message.  If false (the
defaut), it will specify unknown for that file system, provide the error and
continue to check the remaining file systems as expected.
* The `--human-readable` (False by default) option determines if you prefer
to display sizes of different drives in a human format. (Like df Unix/linux
command.)

## Configuration

### Asset registration

[Sensu Assets][4] are the best way to make use of this plugin. If you're not
using an asset, please consider doing so! If you're using sensuctl 5.13 with
Sensu Backend 5.13 or later, you can use the following command to add the asset:

```
sensuctl asset add sensu/check-disk-usage
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][3].

### Check definition

#### Linux example

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: check-disk-usage
  namespace: default
spec:
  command: >-
    check-disk-usage
    --include-fs-type "xfs,ext4"
    --exclude-fs-path "/boot"
    --warning 90
    --critical 95
  subscriptions:
  - system
  runtime_assets:
  - sensu/check-disk-usage
```

#### Windows example
```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: check-disk-usage
  namespace: default
spec:
  command: >-
    check-disk-usage
    --include-fs-type "NTFS"
    --exclude-fs-path "C:,D:"
    --warning 90
    --critical 95
  subscriptions:
  - system
  runtime_assets:
  - sensu/check-disk-usage
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an
Asset. If you would like to compile and install the plugin from source or
contribute to it, download the latest version or create an executable from this
source.

From the local path of the check-disk-usage repository:

```
go build
```

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[3]: https://bonsai.sensu.io/assets/sensu/check-disk-usage
[4]: https://docs.sensu.io/sensu-go/latest/reference/assets/
