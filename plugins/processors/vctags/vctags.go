// This file defines vctags processor plugin for telegraf
//  We use StreamingProcessor in other to make use of goroutines to get data
// from vSphere and update the cache used to populate metrics
//
// Author: Tesifonte Belda
// License: GNU-GPL3 license

package vctags

import (
	"context"
	"fmt"
	"os"
	"net/url"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/processors"
)

type vcTags struct {
	tls.ClientConfig
	VCenter       string          `toml:"vcenter"`
	Username      string          `toml:"username"`
	Password      string          `toml:"password"`
	Timeout       config.Duration `toml:"timeout"`
	VcCategories  []string        `toml:"vsphere_categories"`
	MoIdTag       string          `toml:"metric_moid_tag"`
	CacheInterval config.Duration `toml:"cache_interval"`
	Debug         bool            `toml:"debug"`

	url         *url.URL
	cache       *VcTagCache
	cacheCtx    context.Context
	cacheCancel context.CancelFunc
}

var sampleConfig = `
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
`

// init initializes shim with vcstags processor by importing from main
func init() {
	processors.AddStreaming("vctags", func() telegraf.StreamingProcessor {
		return &vcTags{
			VCenter:       "https://vcenter.local/sdk",
			Username:      "user@corp.local",
			Password:      "secret",
			Timeout:       config.Duration(time.Minute * 3),
			VcCategories:  []string{},
			MoIdTag:       "moid",
			CacheInterval: config.Duration(time.Minute * 10),
		}
	})
}

// Init parses configuration and starts vSphere tags cache
func (p *vcTags) Init() error {
	var err error

	p.url, err = vcPaseURL(p.VCenter, p.Username, p.Password)
	if err != nil {
		return err
	}

	p.cache, err = NewCache(p.url, p.TLSCA, p.InsecureSkipVerify, time.Duration(p.Timeout))
	if err != nil {
		return err
	}
	p.cache.SetCategoryFilter(p.VcCategories)
	p.cache.SetDebug(p.Debug)

	return nil
}

// Start starts this shim and vSphere tags cache goroutine
func (p *vcTags) Start(acc telegraf.Accumulator) error {
	p.cacheCtx, p.cacheCancel = context.WithCancel(context.Background())
	go p.cache.Run(p.cacheCtx, time.Duration(p.CacheInterval))

	return nil
}

// Stop stops this shim and vSphere tags cache goroutine
func (p *vcTags) Stop() error {
	p.cacheCancel()
	return nil
}

// Add applies tags to incoming metrics based on selected vSphere tags/categories
func (p *vcTags) Add(m telegraf.Metric, acc telegraf.Accumulator) error {
	var (
		moid string
		ok   bool
		tags map[string]string
	)

	moid, ok = m.GetTag(p.MoIdTag)
	if ok {
		tags, ok = p.cache.Get(moid)
		if ok {
			for cat, tag := range tags {
				m.AddTag(cat, tag)
				if p.Debug {
					fmt.Fprintf(os.Stderr, "DEBUG enriched metric for %s = %s with tag %s\n", p.MoIdTag, moid, cat)
				}
			}
		}
	} else {
		if p.Debug {
			fmt.Fprintf(os.Stderr, "DEBUG metric with name %s did not have %s tag\n", m.Name(), p.MoIdTag)
		}
	}
	acc.AddMetric(m)

	return nil
}

// SampleConfig shows vctags sample configuration
func (p *vcTags) SampleConfig() string {
	return sampleConfig
}

// Description shows vctags telegraf plugin description
func (p *vcTags) Description() string {
	return "Adds vSphere object's tags to incoming metrics"
}
