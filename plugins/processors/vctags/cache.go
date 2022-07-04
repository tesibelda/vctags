// This file contains vctags plugin package with cache definitions
//
// Author: Tesifonte Belda
// License: GNU-GPL3 license

package vctags

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/kpango/glg"
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
	logger        *glg.Glg
}

// NewCache creates a new cache instance for vSphere objects tags
func NewCache(u *url.URL, s bool, t time.Duration, l *glg.Glg) (*VcTagCache, error) {
	if u == nil {
		return nil, fmt.Errorf(Error_URLNil)
	}
	if l == nil {
		return nil, fmt.Errorf("no logger was provided")
	}
	n := &VcTagCache{
		cache:        gocache.NewCache().WithMaxSize(0),
		urlp:         u,
		insecureSkip: s,
		timeout:      t,
		logger:       l,
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
		return fmt.Errorf("getting categories: %s", err)
	}
	c.logger.Debugf("got categories list, it is %d long", len(cats)) //nolint no error

	vms, err := vcGetVMList(ctxq, c.invClient)
	if err != nil {
		return fmt.Errorf("getting VM list: %s", err)
	}
	c.logger.Debugf("got vm list, it is %d long", len(vms)) //nolint no error

	refs := make([]mo.Reference, len(vms))
	for i := range vms {
		refs[i] = vms[i].Reference()
	}
	vmatagsList, err := vcGetMoListTags(ctxq, m, refs)
	if err != nil {
		return fmt.Errorf("getting VM tags: %s", err)
	}
	c.logger.Debugf("got vm tags list, it is %d long", len(vmatagsList)) //nolint no error

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
	c.logger.Debugf("got vm tags, now cache is %d long", c.cache.Count()) //nolint no error

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
			c.invClient = nil
			return err
		}
		err = c.logger.Infof("created new soap session with vCenter %s", c.urlp.Host)
		if err != nil {
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
		err = c.logger.Infof("new rest session opened with vCenter %s", c.urlp.Host)
		if err != nil {
			return err
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
	if c.cache == nil || c.logger == nil {
		return
	}

	doRun := func() {
		if err := c.populateCache(ctx); err != nil {
			c.logger.Errorf("gathering vSphere tags: %s", err) //nolint no error
		}
	}

	if c.cache.Count() == 0 {
		doRun()
	}

	t := time.NewTicker(pollInterval)
	defer t.Stop()
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
			doRun()
		}
	}
}

// SetCategoryFilter allows configuring a filter of tag categories to read from vSphere
func (c *VcTagCache) SetCategoryFilter(cats []string) {
	c.categoyFilter = cats
}
