// This file contains vctags plugin package with vsphere helper functions
//
// Author: Tesifonte Belda
// License: GNU-GPL3 license

package vctags

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// Common errors raised by vctags
const (
	Error_NoClient   = "No vCenter client, please open a session"
	Error_URLParsing = "Error parsing URL for vcenter"
	Error_URLNil     = "vcenter URL should not be nil"
	Error_NotVC      = "Endpoint does not look like a vCenter"
)

// vcNewClient creates a vSphere vim25.Client for inventory queries
func vcNewClient(
	ctx context.Context,
	u *url.URL,
	skip bool,
) (*govmomi.Client, error) {
	var err error

	if u == nil || u.User == nil {
		return nil, fmt.Errorf(Error_URLNil)
	}

	c, err := govmomi.NewClient(ctx, u, skip)
	if err != nil {
		return nil, err
	}

	if !c.Client.IsVC() {
		return nil, fmt.Errorf(Error_NotVC)
	}

	return c, nil
}

// vcNewRestClient creates a vSphere rest.Client for tags queries
func vcNewRestClient(
	ctx context.Context,
	u *url.URL,
	skip bool,
	c *govmomi.Client,
) (*rest.Client, error) {
	if c == nil || c.Client == nil {
		return nil, fmt.Errorf(Error_NoClient)
	}
	// Share govc's session cache
	s := &cache.Session{
		URL:      u,
		Insecure: skip,
	}

	rc := rest.NewClient(c.Client)
	err := s.Login(ctx, rc, nil)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

// vcCloseClient closes govmomi client
func vcCloseClient(ctx context.Context, c *govmomi.Client) {
	if c != nil {
		_ = c.Logout(ctx) //nolint: no worries for logout errors
	}
}

// vcCloseRestClient closes vSphere rest client
func vcCloseRestClient(ctx context.Context, rc *rest.Client) {
	if rc != nil {
		_ = rc.Logout(ctx) //nolint: no worries for logout errors
	}
}

// vcPaseURL parses vcenter URL params
func vcPaseURL(vcenterUrl, user, pass string) (*url.URL, error) {
	u, err := soap.ParseURL(vcenterUrl)
	if err != nil {
		return nil, fmt.Errorf(string(Error_URLParsing + ": %w" + err.Error()))
	}
	if u == nil {
		return nil, fmt.Errorf(string(Error_URLParsing + ": returned nil"))
	}
	u.User = url.UserPassword(user, pass)

	return u, nil
}

// vcGetVMList return the list of vms
func vcGetVMList(ctx context.Context, c *govmomi.Client) ([]types.ManagedObjectReference, error) {
	if c == nil || c.Client == nil {
		return nil, fmt.Errorf(Error_NoClient)
	}

	v, err := view.NewManager(c.Client).CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}

	vms, err := v.Find(ctx, nil, property.Filter{}) // List all VMs in the inventory
	if err != nil {
		return nil, err
	}

	return vms, err
}

// vcFilterCats gets vSphere tag category list matching input slice of category names
func vcFilterCats(ctx context.Context, mgr *tags.Manager, catnames []string) ([]tags.Category, error) {
	var dofilter bool

	if mgr == nil {
		return nil, fmt.Errorf("No vSphere tag manager, please create one")
	}

	categories, err := mgr.GetCategories(ctx)
	if err != nil {
		return nil, err
	}
	var ocats []tags.Category
	if len(catnames) > 0 {
		dofilter = true
	}
	for _, category := range categories {
		if dofilter {
			if isElementExist(catnames, category.Name) {
				ocats = append(ocats, category)
			}
		} else {
			ocats = append(ocats, category)
		}
	}

	return ocats, nil
}

// vcGetMoListTags returns vSphere tag values of each of the given managed object reference list
func vcGetMoListTags(
	ctx context.Context,
	mgr *tags.Manager,
	refs []mo.Reference,
) ([]tags.AttachedTags, error) {
	attached, err := mgr.GetAttachedTagsOnObjects(ctx, refs)
	if err != nil {
		return nil, err
	}

	return attached, nil
}

// vcIsActive returns true if the vCenter connection is active
func vcIsActive(ctx context.Context, c *govmomi.Client) bool {
	var (
		err error
		ok  bool
	)

	if c == nil || !c.Client.Valid() {
		return false
	}

	ok, err = c.SessionManager.SessionIsActive(ctx)
	if err != nil {
		// skip permission denied error for SessionIsActive call
		if strings.Contains(err.Error(), "Permission") {
			return true
		} else {
			return false
		}
	}

	return ok
}

// isElementExist returns true if a string slice contains a given string
func isElementExist(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
