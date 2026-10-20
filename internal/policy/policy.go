package policy

import "slices"

type Identity struct {
	UserID           int64
	Roles            []string
	Teams            []string
	OIDCGroups       []string
	SystemAdmin      bool
	AdminOwnerGroups []int64
	AdminCategories  []int64
}

type Link struct {
	ID           int64
	OwnerGroupID int64
	CategoryID   int64
	Roles        []string
	Teams        []string
	Enabled      bool
	Deleted      bool
}

func Visible(identity Identity, link Link) bool {
	if !link.Enabled || link.Deleted {
		return false
	}
	if CanManage(identity, link) {
		return true
	}
	return intersectsOrEmpty(identity.Roles, link.Roles) && intersectsOrEmpty(identity.Teams, link.Teams)
}

func CanManage(identity Identity, link Link) bool {
	return identity.SystemAdmin || slices.Contains(identity.AdminOwnerGroups, link.OwnerGroupID) || slices.Contains(identity.AdminCategories, link.CategoryID)
}

func CanCreate(identity Identity, ownerGroupID int64) bool {
	return identity.SystemAdmin || slices.Contains(identity.AdminOwnerGroups, ownerGroupID)
}

func intersectsOrEmpty(have, required []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, r := range required {
		if slices.Contains(have, r) {
			return true
		}
	}
	return false
}
