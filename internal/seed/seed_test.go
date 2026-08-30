package seed

import "testing"

func TestValidate(t *testing.T) {
	d := Document{Version: 1, Categories: []Category{{Slug: "dev", Name: "Development"}}, Groups: []Group{{Slug: "platform", Name: "Platform"}}, Links: []Link{{Slug: "docs", Title: "Docs", URL: "https://docs.example", Category: "dev", OwnerGroup: "platform"}}}
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
	d.Links[0].URL = "javascript:alert(1)"
	if err := d.Validate(); err == nil {
		t.Fatal("expected blocked scheme")
	}
}

func TestParse(t *testing.T) {
	document, err := Parse([]byte("version: 1\ncategories:\n  - slug: dev\n    name: Development\ngroups:\n  - slug: platform\n    name: Platform\nlinks:\n  - slug: docs\n    title: Docs\n    url: https://docs.example\n    category: dev\n    ownerGroup: platform\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Links) != 1 || document.Links[0].Slug != "docs" {
		t.Fatalf("unexpected document: %#v", document)
	}
	if _, err = Parse([]byte("version: 1\nlinks: [")); err == nil {
		t.Fatal("expected malformed YAML to fail")
	}
}
