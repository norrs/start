package seed

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type Document struct {
	Version    int        `yaml:"version"`
	Categories []Category `yaml:"categories"`
	Groups     []Group    `yaml:"groups"`
	Links      []Link     `yaml:"links"`
}
type Category struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Color       string `yaml:"color"`
	Order       int    `yaml:"order"`
}
type Group struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}
type Link struct {
	Slug        string   `yaml:"slug"`
	Title       string   `yaml:"title"`
	URL         string   `yaml:"url"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category"`
	OwnerGroup  string   `yaml:"ownerGroup"`
	Icon        Icon     `yaml:"icon"`
	Tags        []string `yaml:"tags"`
	Audience    Audience `yaml:"audience"`
}
type Icon struct {
	Provider string `yaml:"provider"`
	Name     string `yaml:"name"`
	Color    string `yaml:"color"`
}
type Audience struct {
	Roles []string `yaml:"roles"`
	Teams []string `yaml:"teams"`
}

func Load(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	return Parse(b)
}

// Parse decodes and validates a YAML catalog document.
func Parse(b []byte) (Document, error) {
	var d Document
	if err := yaml.Unmarshal(b, &d); err != nil {
		return Document{}, fmt.Errorf("decode seed: %w", err)
	}
	if err := d.Validate(); err != nil {
		return Document{}, err
	}
	return d, nil
}

func (d Document) Validate() error {
	if d.Version != 1 {
		return fmt.Errorf("unsupported seed version %d", d.Version)
	}
	categories, groups, links := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, c := range d.Categories {
		if err := uniqueSlug("category", c.Slug, categories); err != nil {
			return err
		}
		if c.Name == "" {
			return errors.New("category name is required")
		}
	}
	for _, g := range d.Groups {
		if err := uniqueSlug("group", g.Slug, groups); err != nil {
			return err
		}
		if g.Name == "" {
			return errors.New("group name is required")
		}
	}
	for _, l := range d.Links {
		if err := uniqueSlug("link", l.Slug, links); err != nil {
			return err
		}
		if l.Title == "" {
			return fmt.Errorf("link %s title is required", l.Slug)
		}
		u, err := url.Parse(l.URL)
		if err != nil || u.Scheme == "" {
			return fmt.Errorf("link %s has invalid URL", l.Slug)
		}
		switch strings.ToLower(u.Scheme) {
		case "javascript", "data", "vbscript":
			return fmt.Errorf("link %s uses blocked URL scheme", l.Slug)
		}
		if !categories[l.Category] {
			return fmt.Errorf("link %s references unknown category %s", l.Slug, l.Category)
		}
		if !groups[l.OwnerGroup] {
			return fmt.Errorf("link %s references unknown owner group %s", l.Slug, l.OwnerGroup)
		}
	}
	return nil
}

func uniqueSlug(kind, slug string, seen map[string]bool) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid %s slug %q", kind, slug)
	}
	if seen[slug] {
		return fmt.Errorf("duplicate %s slug %q", kind, slug)
	}
	seen[slug] = true
	return nil
}
