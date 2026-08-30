package catalog

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/norrs/start/db/migrations"
	"github.com/norrs/start/internal/policy"
	"github.com/norrs/start/internal/seed"
	_ "modernc.org/sqlite"
)

func TestStoreLifecycleAndIdentityMappings(t *testing.T) {
	db, err := sql.Open("sqlite", "file:catalog-test?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err = migrations.Up(db); err != nil {
		t.Fatal(err)
	}
	store := New(db)
	doc := seed.Document{Version: 1, Categories: []seed.Category{{Slug: "development", Name: "Development", Color: "#123456"}}, Groups: []seed.Group{{Slug: "platform", Name: "Platform"}}, Links: []seed.Link{{Slug: "docs", Title: "Docs", URL: "https://docs.example", Category: "development", OwnerGroup: "platform"}}}
	if err = doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if err = store.ImportSeedIfEmpty(context.Background(), doc); err != nil {
		t.Fatal(err)
	}

	var roleID, teamID int64
	if err = db.QueryRow(`INSERT INTO roles(slug,name)VALUES('developer','Developer') RETURNING id`).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`INSERT INTO teams(slug,name)VALUES('platform','Platform') RETURNING id`).Scan(&teamID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO identity_group_mappings(group_value,target_type,target_id)VALUES('start-users','role',?),('start-users','team',?)`, roleID, teamID); err != nil {
		t.Fatal(err)
	}
	u, err := store.EnsureUser(context.Background(), "https://id.example", "alice", "alice@example.com", "Alice", []string{"start-users", "start-admins"})
	if err != nil {
		t.Fatal(err)
	}
	identity, user, err := store.Identity(context.Background(), u.ID, []string{"start-admins"})
	if err != nil {
		t.Fatal(err)
	}
	if !identity.SystemAdmin || len(identity.Roles) != 1 || identity.Roles[0] != "developer" || len(identity.Teams) != 1 || identity.Teams[0] != "platform" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
	boot, err := store.Bootstrap(context.Background(), identity, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(boot.Links) != 1 || !boot.Links[0].CanManage {
		t.Fatalf("unexpected bootstrap: %#v", boot.Links)
	}

	input := LinkInput{Slug: "status", Title: "Status", URL: "http://status.internal", CategoryID: boot.Categories[0].ID, OwnerGroupID: boot.OwnerGroups[0].ID, Tags: []string{"monitoring"}, Enabled: true}
	linkID, err := store.CreateLink(context.Background(), identity, input, "request-create")
	if err != nil {
		t.Fatal(err)
	}
	input.Title = "Service Status"
	input.Revision = 1
	if err = store.UpdateLink(context.Background(), identity, linkID, input, "request-update"); err != nil {
		t.Fatal(err)
	}
	if err = store.DeleteLink(context.Background(), identity, linkID, 2, "request-delete"); err != nil {
		t.Fatal(err)
	}
	revisions, err := store.Revisions(context.Background(), identity, linkID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 3 || revisions[0].Action != "delete" {
		t.Fatalf("unexpected revisions: %#v", revisions)
	}
	if err = store.Restore(context.Background(), identity, linkID, 2, "request-restore"); err != nil {
		t.Fatal(err)
	}
	boot, err = store.Bootstrap(context.Background(), identity, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(boot.Links) != 2 {
		t.Fatalf("expected restored link, got %d", len(boot.Links))
	}
	var audits int
	if err = db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE target_id=?`, linkID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 4 {
		t.Fatalf("expected 4 audit events, got %d", audits)
	}

	second := LinkInput{Slug: "runbooks", Title: "Runbooks", URL: "https://runbooks.example", CategoryID: boot.Categories[0].ID, OwnerGroupID: boot.OwnerGroups[0].ID, Enabled: true}
	secondID, err := store.CreateLink(context.Background(), identity, second, "request-second-create")
	if err != nil {
		t.Fatal(err)
	}
	second.Title = "Platform Runbooks"
	second.Revision = 1
	if err = store.UpdateLink(context.Background(), identity, secondID, second, "request-second-update"); err != nil {
		t.Fatal(err)
	}
	history, err := store.LinkHistory(context.Background(), identity)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 7 {
		t.Fatalf("expected all 7 link changes in history, got %d", len(history))
	}
	if err = store.RestoreMany(context.Background(), identity, []RestoreTarget{
		{LinkID: linkID, Revision: 1, CurrentRevision: 4},
		{LinkID: secondID, Revision: 1, CurrentRevision: 2},
	}, "request-batch-restore"); err != nil {
		t.Fatal(err)
	}
	var firstTitle, secondTitle string
	var firstRevision, secondRevision int64
	if err = db.QueryRow(`SELECT title, revision FROM links WHERE id=?`, linkID).Scan(&firstTitle, &firstRevision); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT title, revision FROM links WHERE id=?`, secondID).Scan(&secondTitle, &secondRevision); err != nil {
		t.Fatal(err)
	}
	if firstTitle != "Status" || secondTitle != "Runbooks" || firstRevision != 5 || secondRevision != 3 {
		t.Fatalf("unexpected selective restore result: %q r%d, %q r%d", firstTitle, firstRevision, secondTitle, secondRevision)
	}
	if err = store.RestoreMany(context.Background(), identity, []RestoreTarget{
		{LinkID: linkID, Revision: 2, CurrentRevision: 5},
		{LinkID: secondID, Revision: 2, CurrentRevision: 2},
	}, "request-stale-batch"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected atomic stale conflict, got %v", err)
	}
	if err = db.QueryRow(`SELECT revision FROM links WHERE id=?`, linkID).Scan(&firstRevision); err != nil {
		t.Fatal(err)
	}
	if firstRevision != 5 {
		t.Fatalf("stale batch partially committed first link at revision %d", firstRevision)
	}
}

func TestGroupAdminCannotCreateOutsideOwnerGroup(t *testing.T) {
	id := policy.Identity{UserID: 1, AdminOwnerGroups: []int64{7}}
	if !policy.CanCreate(id, 7) || policy.CanCreate(id, 8) {
		t.Fatal("owner-group create policy failed")
	}
}

func TestImportSeedMergesPopulatedCatalogAndTracksHistory(t *testing.T) {
	database, err := sql.Open("sqlite", "file:catalog-import-test?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	if err = migrations.Up(database); err != nil {
		t.Fatal(err)
	}
	store := New(database)
	initial := seed.Document{
		Version:    1,
		Categories: []seed.Category{{Slug: "development", Name: "Development"}},
		Groups:     []seed.Group{{Slug: "platform", Name: "Platform"}},
		Links: []seed.Link{
			{Slug: "docs", Title: "Docs", URL: "https://docs.example", Category: "development", OwnerGroup: "platform", Tags: []string{"reference"}},
			{Slug: "keep", Title: "Keep me", URL: "https://keep.example", Category: "development", OwnerGroup: "platform"},
		},
	}
	if err = store.ImportSeedIfEmpty(context.Background(), initial); err != nil {
		t.Fatal(err)
	}
	var docsID int64
	if err = database.QueryRow(`SELECT id FROM links WHERE slug='docs'`).Scan(&docsID); err != nil {
		t.Fatal(err)
	}
	user, err := store.EnsureUser(context.Background(), "https://id.example", "import-admin", "import@example.com", "Import Admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.DeleteLink(context.Background(), policy.Identity{UserID: user.ID, SystemAdmin: true}, docsID, 1, "delete-before-import"); err != nil {
		t.Fatal(err)
	}

	merged := seed.Document{
		Version:    1,
		Categories: []seed.Category{{Slug: "development", Name: "Engineering", Color: "#123456"}},
		Groups:     []seed.Group{{Slug: "platform", Name: "Platform Team"}},
		Links: []seed.Link{
			{Slug: "docs", Title: "Engineering docs", URL: "https://docs.example/new", Category: "development", OwnerGroup: "platform", Tags: []string{"Reference", "Go"}},
			{Slug: "status", Title: "Status", URL: "https://status.example", Category: "development", OwnerGroup: "platform"},
		},
	}
	result, err := store.ImportSeed(context.Background(), merged, "merge-import")
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 1 || result.Restored != 1 || result.Updated != 0 {
		t.Fatalf("unexpected merge result: %#v", result)
	}
	var title string
	var revision int64
	var deleted sql.NullString
	if err = database.QueryRow(`SELECT title,revision,deleted_at FROM links WHERE id=?`, docsID).Scan(&title, &revision, &deleted); err != nil {
		t.Fatal(err)
	}
	if title != "Engineering docs" || revision != 3 || deleted.Valid {
		t.Fatalf("unexpected restored link: title=%q revision=%d deleted=%v", title, revision, deleted.Valid)
	}
	var linkCount, keepCount int
	if err = database.QueryRow(`SELECT COUNT(*),COUNT(*) FILTER (WHERE slug='keep') FROM links`).Scan(&linkCount, &keepCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 3 || keepCount != 1 {
		t.Fatalf("merge changed unrelated records: links=%d keep=%d", linkCount, keepCount)
	}
	result, err = store.ImportSeed(context.Background(), merged, "repeat-import")
	if err != nil {
		t.Fatal(err)
	}
	if result.Unchanged != 2 || result.Created+result.Updated+result.Restored != 0 {
		t.Fatalf("repeated import was not idempotent: %#v", result)
	}
}
