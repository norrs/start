package policy

import "testing"

func TestVisibilityDimensions(t *testing.T) {
	id := Identity{Roles: []string{"developer"}, Teams: []string{"platform"}}
	cases := []struct {
		name string
		link Link
		want bool
	}{
		{"public", Link{Enabled: true}, true},
		{"role", Link{Enabled: true, Roles: []string{"developer"}}, true},
		{"both", Link{Enabled: true, Roles: []string{"developer"}, Teams: []string{"platform"}}, true},
		{"wrong team", Link{Enabled: true, Roles: []string{"developer"}, Teams: []string{"media"}}, false},
		{"disabled", Link{}, false},
		{"deleted", Link{Enabled: true, Deleted: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Visible(id, tc.link); got != tc.want {
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
