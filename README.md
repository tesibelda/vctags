# vctags

vctags is an execd processor plugin for [Telegraf](https://github.com/influxdata/telegraf) that populates metrics with selected tags from VMware vSphere objects (using [govmomi library](https://github.com/vmware/govmomi/)). Currently only VirtualMachine objects are supported.

[![License: GPL v3](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/tesibelda/vctags)](https://goreportcard.com/report/github.com/tesibelda/vctags)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/tesibelda/vctags?display_name=release)

# Compatibility

Current releases are built with a govmomi library version that supports vCenter 6.5, 6.7 and 7.0.
Use telegraf v1.15 or above so that execd processor is available. 

# Configuration

* Download the [latest release package](https://github.com/tesibelda/vctags/releases/latest) for your platform.

* Edit vctags.conf file as needed. Example:

```toml
[[processors.vctags]]
  ## vCenter URL to be monitored and its credential
  vcenter = "https://vcenter.local/sdk"
  username = "user@corp.local"
  password = "secret"
  ## total vSphere requests timeout
  # timeout = "3m"
  ## Optional TLS CA full file path
  # tls_ca = ""
  ## Use SSL but skip chain & host verification
  # insecure_skip_verify = false

  ## List of vSphere tag categories to populate metrics
  # vsphere_categories = []
  ## Metric's tag to identify vSphere managed object Id
  # metric_moid_tag = "moid"
  ## vSphere tag cache refresh interval
  # cache_interval = "10m"
  ## Enable debug
  # debug = falss
```

* Edit telegraf's execd processor configuration as needed. Example:

```
## Gather vSphere vCenter status and basic stats
[[processors.execd]]
  command = ["/path/to/vctags_binary", "-config", "/path/to/vctags.conf"]
```

* Restart or reload Telegraf.


# Example output

Metrics will have environment tag added to them by using vsphere_categories=\["environment"\]:
```plain
vsphere_vm_cpu,clustername=DC0_C0,environment=PRE,esxhostname=DC0_C0_H0,guest=other,host=host.example.com,moid=vm-44,os=Mac,source=DC0_C0_RP0_VM1,vcenter=localhost:8989,vmname=DC0_C0_RP0_VM1 demand_average=328i,run_summation=3481i,ready_summation=122i,usage_average=7.95,used_summation=2167i 1535660339000000000
vsphere_vm_net,clustername=DC0_C0,environment=PRE,esxhostname=DC0_C0_H0,guest=other,host=host.example.com,moid=vm-44,os=Mac,source=DC0_C0_RP0_VM1,vcenter=localhost:8989,vmname=DC0_C0_RP0_VM1 bytesTx_average=282i,bytesRx_average=196i 1535660339000000000
vsphere_vm_virtualDisk,clustername=DC0_C0,environment=PRE,esxhostname=DC0_C0_H0,guest=other,host=host.example.com,moid=vm-44,os=Mac,source=DC0_C0_RP0_VM1,vcenter=localhost:8989,vmname=DC0_C0_RP0_VM1 write_average=321i,read_average=13i 1535660339000000000
```
A tag category called 'environment' with tag name set as 'PRE' was previously configured for DC0_C0_RP0_VM1 VM.

# Build Instructions

Download the repo

    $ git clone git@github.com:tesibelda/vctags.git

build the "vctags" binary

    $ go build -o bin/vctags cmd/main.go
    
 (if you're using windows, you'll want to give it an .exe extension)
 
    $ go build -o bin\vctags.exe cmd/main.go

 If you use [go-task](https://github.com/go-task/task) execute one of these
 
    $ task linux:build
	$ task windows:build

# Author

Tesifonte Belda (https://github.com/tesibelda)

# Disclaimer

The author and maintainers are not affiliated with VMware.
VMware is a registered trademark or trademark of VMware, Inc.

# License

[GNU-GPL3 License](https://github.com/tesibelda/vctags/blob/master/LICENSE)
