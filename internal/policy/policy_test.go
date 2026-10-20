package policy

import "testing"

func TestVisibilityDimensions(t *testing.T) {
	cases := []struct {
		name     string
		identity Identity
		link     Link
		want     bool
	}{
		{"public", Identity{}, Link{Enabled: true}, true},
		{"role", Identity{Roles: []string{"developer"}}, Link{Enabled: true, Roles: []string{"developer"}}, true},
		{"both", Identity{Roles: []string{"developer"}, Teams: []string{"platform"}}, Link{Enabled: true, Roles: []string{"developer"}, Teams: []string{"platform"}}, true},
		{"wrong team", Identity{Roles: []string{"developer"}, Teams: []string{"platform"}}, Link{Enabled: true, Roles: []string{"developer"}, Teams: []string{"media"}}, false},
		{"system admin", Identity{SystemAdmin: true}, Link{Enabled: true, Roles: []string{"media"}}, true},
		{"owner group admin", Identity{AdminOwnerGroups: []int64{4}}, Link{Enabled: true, OwnerGroupID: 4, Roles: []string{"media"}}, true},
		{"category admin", Identity{AdminCategories: []int64{7}}, Link{Enabled: true, CategoryID: 7, Roles: []string{"media"}}, true},
		{"unrelated group admin", Identity{AdminOwnerGroups: []int64{5}}, Link{Enabled: true, OwnerGroupID: 4, Roles: []string{"media"}}, false},
		{"disabled admin link", Identity{SystemAdmin: true}, Link{}, false},
		{"deleted admin link", Identity{SystemAdmin: true}, Link{Enabled: true, Deleted: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Visible(tc.identity, tc.link); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCreateRequiresOwnerGroup(t *testing.T) {
	id := Identity{AdminCategories: []int64{2}, AdminOwnerGroups: []int64{4}}
	if CanCreate(id, 2) {
		t.Fatal("category administration must not grant create")
	}
	if !CanCreate(id, 4) {
		t.Fatal("owner group administration should grant create")
	}
}
