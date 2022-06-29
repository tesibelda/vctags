// This file contains vctags plugin package with cache definitions
//
// Author: Tesifonte Belda
// License: GNU-GPL3 license

package vctags

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/TwiN/gocache/v2"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
)

type VcTagCache struct {
	cache         *gocache.Cache
	urlp          *url.URL
	insecureSkip  bool
	timeout       time.Duration
	categoyFilter []string
	invClient     *govmomi.Client
	restClient    *rest.Client
	debug         bool
}

// NewCache creates a new cache instance for vSphere objects tags
func NewCache(u *url.URL, skip bool, t time.Duration) (*VcTagCache, error) {
	if u == nil {
		return nil, fmt.Errorf(Error_URLNil)
	}
	n := &VcTagCache{
		cache:        gocache.NewCache().WithMaxSize(0),
		urlp:         u,
		insecureSkip: skip,
		timeout:      t,
	}
	return n, nil
}

// populateCache populates the cache with selected tags from vSphere objects
//  currently only VMs
func (c *VcTagCache) populateCache(ctx context.Context) error {
	var err error

	if err = c.keepSessionsAlive(ctx); err != nil {
		return err
	}

	ctxq, cancelq := context.WithTimeout(ctx, time.Duration(c.timeout))
	defer cancelq()

	m := tags.NewManager(c.restClient)
	cats, err := vcFilterCats(ctxq, m, c.categoyFilter)
	if err != nil {
		vcCloseRestClient(ctx, c.restClient)
		c.restClient = nil
		return err
	}
	if c.debug {
		fmt.Fprintf(os.Stderr, "DEBUG got categories list, it is %d long\n", len(cats))
	}

	vms, err := vcGetVMList(ctxq, c.invClient)
	if err != nil {
		return err
	}
	if c.debug {
		fmt.Fprintf(os.Stderr, "DEBUG got vm list, it is %d long\n", len(vms))
	}

	refs := make([]mo.Reference, len(vms))
	for i := range vms {
		refs[i] = vms[i].Reference()
	}
	vmatagsList, err := vcGetMoListTags(ctxq, m, refs)
	if err != nil {
		return err
	}
	if c.debug {
		fmt.Fprintf(os.Stderr, "DEBUG got vm tags list, it is %d long\n", len(vmatagsList))
	}

	c.cache.Clear()
	for _, vmatags := range vmatagsList {
		var mtags = make(map[string]string, len(vmatags.Tags))
		for _, vmtag := range vmatags.Tags {
			for _, cat := range cats {
				if cat.ID == vmtag.CategoryID {
					mtags[cat.Name] = vmtag.Name
				}
			}
		}
		if len(mtags) > 0 {
			c.cache.Set(vmatags.ObjectID.Reference().Value, mtags)
		}
	}
	if c.debug {
		fmt.Fprintf(os.Stderr, "DEBUG got vm tags, now cache is %d long\n", c.cache.Count())
	}

	return err
}

// keepSessionsAlive keeps vCenter sessions alive
func (c *VcTagCache) keepSessionsAlive(ctx context.Context) error {
	err := c.keepSoapSessionAlive(ctx)
	if err != nil {
		return err
	}

	return c.keepRestSessionAlive(ctx)
}

// keepSoapSessionAlive keeps vCenter soap session alive
func (c *VcTagCache) keepSoapSessionAlive(ctx context.Context) error {
	if c.invClient == nil || !vcSoapIsActive(ctx, c.invClient) {
		var err error
		c.invClient, err = vcNewClient(ctx, c.urlp, c.insecureSkip)
		if err != nil {
			if c.debug {
				fmt.Fprintf(os.Stderr, "DEBUG created new soap session with vCenter %s\n", c.urlp.Host)
			}
			c.invClient = nil
			return err
		}
	}

	return nil
}

// keepRestSessionAlive tries to keep vCenter rest session alive
func (c *VcTagCache) keepRestSessionAlive(ctx context.Context) error {
	if c.restClient == nil || !vcRestIsActive(ctx, c.restClient) {
		var err error
		c.restClient, err = vcNewRestClient(ctx, c.urlp, c.insecureSkip, c.invClient)
		if err != nil {
			c.restClient = nil
			return err
		}
		if c.debug {
			fmt.Fprintf(os.Stderr, "DEBUG new rest session opened with vCenter %s\n", c.urlp.Host)
		}
	}

	return nil
}


// Get returns tags from the cache corresponding to the given moid
func (c *VcTagCache) Get(k string) (map[string]string, bool) {
	if c.cache == nil {
		return nil, false
	}
	t, e := c.cache.Get(k)
	if t == nil {
		return nil, false
	}
	return t.(map[string]string), e
}

// Run executes a permanent loop waiting for context end or cache refresh triger
func (c *VcTagCache) Run(ctx context.Context, pollInterval time.Duration) {
	if c.cache == nil {
		return
	}

	if c.cache.Count() == 0 {
		if err := c.populateCache(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR gathering vSphere tags: %s\n", err)
		}
	}

	t := time.NewTicker(pollInterval)
	defer c.cache.Clear()
	for {
		// see what's up
		select {
		case <-ctx.Done():
			vcCloseRestClient(ctx, c.restClient)
			c.restClient = nil
			vcCloseClient(ctx, c.invClient)
			c.invClient = nil
			return
		case <-t.C:
			if err := c.populateCache(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR gathering vSphere tags: %s\n", err)
			}
		}
	}
}

// SetCategoryFilter allows configuring a filter of tag categories to read from vSphere
func (c *VcTagCache) SetCategoryFilter(cats []string) {
	c.categoyFilter = cats
}

// SetCategoryFilter allows configuring a filter of tag categories to read from vSphere
func (c *VcTagCache) SetDebug(db bool) {
	c.debug = db
}
