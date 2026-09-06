package mcp

import "webtyp.com/model"

// AllowAll grants any permission. Development and tests ONLY.
//
// It is a named, greppable escape hatch on purpose: `grep -rn AllowAll` lists every place
// where authorization has been switched off. A silent default that allowed everything would
// be the same thing, with nobody able to find it.
func AllowAll(_ string, _ model.Resource, _ model.Action) bool {
	return true
}

var _ model.Authorizer = AllowAll
