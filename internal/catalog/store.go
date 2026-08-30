package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	dbgen "github.com/norrs/start/internal/db"
	"github.com/norrs/start/internal/policy"
	"github.com/norrs/start/internal/seed"
)

var slugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type Store struct {
	db *sql.DB
	q  *dbgen.Queries
}

func New(db *sql.DB) *Store { return &Store{db: db, q: dbgen.New(db)} }

func (s *Store) EnsureUser(ctx context.Context, issuer, subject, email, name string, groups []string) (dbgen.User, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	u, err := s.q.UpsertUser(ctx, dbgen.UpsertUserParams{Issuer: issuer, Subject: subject, Email: email, DisplayName: name, ClaimsRefreshedAt: now, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return dbgen.User{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return dbgen.User{}, err
	}
	q := s.q.WithTx(tx)
	if err := q.ReplaceUserOIDCGroupsDelete(ctx, u.ID); err != nil {
		_ = tx.Rollback()
		return dbgen.User{}, err
	}
	for _, group := range unique(groups) {
		if err := q.AddUserOIDCGroup(ctx, dbgen.AddUserOIDCGroupParams{UserID: u.ID, GroupValue: group, ClaimSeenAt: now}); err != nil {
			_ = tx.Rollback()
			return dbgen.User{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id=? AND source='oidc'`, u.ID); err != nil {
		_ = tx.Rollback()
		return dbgen.User{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles(user_id,role_id,source,last_seen_at)
		SELECT ?,m.target_id,'oidc',? FROM identity_group_mappings m JOIN roles r ON r.id=m.target_id
		JOIN user_oidc_groups ug ON ug.group_value=m.group_value AND ug.user_id=? WHERE m.target_type='role'`, u.ID, now, u.ID); err != nil {
		_ = tx.Rollback()
		return dbgen.User{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_teams WHERE user_id=? AND source='oidc'`, u.ID); err != nil {
		_ = tx.Rollback()
		return dbgen.User{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_teams(user_id,team_id,source,last_seen_at)
		SELECT ?,m.target_id,'oidc',? FROM identity_group_mappings m JOIN teams t ON t.id=m.target_id
		JOIN user_oidc_groups ug ON ug.group_value=m.group_value AND ug.user_id=? WHERE m.target_type='team'`, u.ID, now, u.ID); err != nil {
		_ = tx.Rollback()
		return dbgen.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return dbgen.User{}, err
	}
	return u, nil
}

func (s *Store) Identity(ctx context.Context, userID int64, bootstrapAdminGroups []string) (policy.Identity, dbgen.User, error) {
	u, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return policy.Identity{}, dbgen.User{}, err
	}
	roles, err := s.q.ListUserRoleSlugs(ctx, userID)
	if err != nil {
		return policy.Identity{}, dbgen.User{}, err
	}
	teams, err := s.q.ListUserTeamSlugs(ctx, userID)
	if err != nil {
		return policy.Identity{}, dbgen.User{}, err
	}
	groups, err := s.q.ListUserOIDCGroups(ctx, userID)
	if err != nil {
		return policy.Identity{}, dbgen.User{}, err
	}
	id := policy.Identity{UserID: userID, Roles: roles, Teams: teams, OIDCGroups: groups}
	for _, g := range groups {
		if slices.Contains(bootstrapAdminGroups, g) {
			id.SystemAdmin = true
		}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT source_type, user_id, oidc_group_value, capability, owner_group_id, category_id FROM admin_grants
		WHERE (source_type='user' AND user_id=?) OR (source_type='oidc_group' AND oidc_group_value IN (SELECT group_value FROM user_oidc_groups WHERE user_id=?))`, userID, userID)
	if err != nil {
		return policy.Identity{}, dbgen.User{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var source, group, capability sql.NullString
		var uid, owner, category sql.NullInt64
		if err := rows.Scan(&source, &uid, &group, &capability, &owner, &category); err != nil {
			return policy.Identity{}, dbgen.User{}, err
		}
		switch capability.String {
		case "system_admin":
			id.SystemAdmin = true
		case "group_admin":
			if owner.Valid {
				id.AdminOwnerGroups = append(id.AdminOwnerGroups, owner.Int64)
			}
		case "category_admin":
			if category.Valid {
				id.AdminCategories = append(id.AdminCategories, category.Int64)
			}
		}
	}
	return id, u, rows.Err()
}

func (s *Store) Bootstrap(ctx context.Context, id policy.Identity, user dbgen.User) (Bootstrap, error) {
	settings, err := s.q.GetAppSettings(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	prefs, err := s.preferences(ctx, user.ID)
	if err != nil {
		return Bootstrap{}, err
	}
	cats, err := s.q.ListCategories(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	groups, err := s.q.ListOwnerGroups(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	hiddenIDs, err := s.q.ListHiddenLinkIDs(ctx, user.ID)
	if err != nil {
		return Bootstrap{}, err
	}
	hidden := map[int64]bool{}
	for _, v := range hiddenIDs {
		hidden[v] = true
	}
	rows, err := s.q.ListActiveLinks(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	out := Bootstrap{
		User:        UserView{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Roles: id.Roles, Teams: id.Teams, Groups: id.OIDCGroups, SystemAdmin: id.SystemAdmin, CanManageLinks: id.SystemAdmin || len(id.AdminOwnerGroups) > 0 || len(id.AdminCategories) > 0},
		Branding:    Branding{AppName: settings.AppName, WelcomeText: settings.WelcomeText, LogoURL: settings.LogoUrl, AccentColor: settings.AccentColor, SupportURL: settings.SupportUrl},
		Preferences: prefs,
		Categories:  make([]CategoryView, 0, len(cats)), OwnerGroups: make([]OwnerGroupView, 0, len(groups)), Links: make([]LinkView, 0, len(rows)),
	}
	for _, c := range cats {
		out.Categories = append(out.Categories, CategoryView{ID: c.ID, Slug: c.Slug, Name: c.Name, Description: c.Description, Color: c.Color, SortOrder: c.SortOrder})
	}
	for _, g := range groups {
		out.OwnerGroups = append(out.OwnerGroups, OwnerGroupView{ID: g.ID, Slug: g.Slug, Name: g.Name, Description: g.Description, CanCreate: policy.CanCreate(id, g.ID)})
	}
	for _, r := range rows {
		tags, e := s.q.ListLinkTags(ctx, r.ID)
		if e != nil {
			return Bootstrap{}, e
		}
		roles, e := s.q.ListLinkRoles(ctx, r.ID)
		if e != nil {
			return Bootstrap{}, e
		}
		teams, e := s.q.ListLinkTeams(ctx, r.ID)
		if e != nil {
			return Bootstrap{}, e
		}
		p := policy.Link{ID: r.ID, OwnerGroupID: r.OwnerGroupID, CategoryID: r.CategoryID, Roles: roles, Teams: teams, Enabled: r.Enabled == 1}
		if !policy.Visible(id, p) {
			continue
		}
		if hidden[r.ID] && !prefs.ShowHidden {
			continue
		}
		out.Links = append(out.Links, LinkView{ID: r.ID, Slug: r.Slug, Title: r.Title, URL: r.Url, Description: r.Description, IconProvider: r.IconProvider, IconName: r.IconName, IconColor: r.IconColor, CategoryID: r.CategoryID, CategorySlug: r.CategorySlug, CategoryName: r.CategoryName, CategoryColor: r.CategoryColor, OwnerGroupID: r.OwnerGroupID, OwnerGroupSlug: r.OwnerGroupSlug, OwnerGroupName: r.OwnerGroupName, Tags: tags, Roles: roles, Teams: teams, Hidden: hidden[r.ID], CanManage: policy.CanManage(id, p), Revision: r.Revision})
	}
	return out, nil
}

func (s *Store) preferences(ctx context.Context, userID int64) (Preferences, error) {
	p, err := s.q.GetPreferences(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		p, err = s.q.UpsertPreferences(ctx, dbgen.UpsertPreferencesParams{UserID: userID, Theme: "dark", ViewMode: "categorized", ShowDescriptions: 1, ShowCategories: 1, ShowIcons: 1, ColoredIcons: 1, ShowHidden: 0, OpenNewTab: 0, HotkeysEnabled: 1, UpdatedAt: now})
	}
	if err != nil {
		return Preferences{}, err
	}
	return prefsView(p), nil
}

func (s *Store) SavePreferences(ctx context.Context, userID int64, p Preferences) (Preferences, error) {
	if !slices.Contains([]string{"dark", "light", "hacker", "1984"}, p.Theme) {
		return Preferences{}, errors.New("invalid theme")
	}
	if !slices.Contains([]string{"categorized", "flat"}, p.ViewMode) {
		return Preferences{}, errors.New("invalid view mode")
	}
	r, err := s.q.UpsertPreferences(ctx, dbgen.UpsertPreferencesParams{UserID: userID, Theme: p.Theme, ViewMode: p.ViewMode, ShowDescriptions: btoi(p.ShowDescriptions), ShowCategories: btoi(p.ShowCategories), ShowIcons: btoi(p.ShowIcons), ColoredIcons: btoi(p.ColoredIcons), ShowHidden: btoi(p.ShowHidden), OpenNewTab: btoi(p.OpenNewTab), HotkeysEnabled: btoi(p.HotkeysEnabled), UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		return Preferences{}, err
	}
	return prefsView(r), nil
}

func (s *Store) Hide(ctx context.Context, userID, linkID int64, hide bool) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if hide {
		return s.q.HideLink(ctx, dbgen.HideLinkParams{UserID: userID, LinkID: linkID, HiddenAt: now})
	}
	return s.q.ShowLink(ctx, dbgen.ShowLinkParams{UserID: userID, LinkID: linkID})
}

func (s *Store) CreateLink(ctx context.Context, id policy.Identity, input LinkInput, requestID string) (int64, error) {
	if err := validateInput(input); err != nil {
		return 0, err
	}
	if !policy.CanCreate(id, input.OwnerGroupID) {
		return 0, ErrForbidden
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, `INSERT INTO links(slug,title,url,description,icon_provider,icon_name,icon_color,category_id,owner_group_id,enabled,revision,created_by,updated_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,1,?,?,?,?)`, input.Slug, input.Title, input.URL, input.Description, input.IconProvider, input.IconName, input.IconColor, input.CategoryID, input.OwnerGroupID, btoi(input.Enabled), id.UserID, id.UserID, now, now)
	if err != nil {
		return 0, err
	}
	linkID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err = s.replaceRelations(ctx, tx, linkID, input); err != nil {
		return 0, err
	}
	if err = s.writeRevision(ctx, tx, linkID, 1, "create", snapshot{Input: input}, id.UserID); err != nil {
		return 0, err
	}
	if err = s.audit(ctx, tx, id.UserID, "link.create", "link", fmt.Sprint(linkID), nil, input, requestID); err != nil {
		return 0, err
	}
	return linkID, tx.Commit()
}

func (s *Store) UpdateLink(ctx context.Context, id policy.Identity, linkID int64, input LinkInput, requestID string) error {
	if err := validateInput(input); err != nil {
		return err
	}
	existing, err := s.q.GetLink(ctx, linkID)
	if err != nil {
		return err
	}
	p := policy.Link{ID: linkID, OwnerGroupID: existing.OwnerGroupID, CategoryID: existing.CategoryID, Enabled: existing.Enabled == 1, Deleted: existing.DeletedAt.Valid}
	if !policy.CanManage(id, p) || (!id.SystemAdmin && existing.OwnerGroupID != input.OwnerGroupID && !slices.Contains(id.AdminOwnerGroups, input.OwnerGroupID)) {
		return ErrForbidden
	}
	if input.Revision != existing.Revision {
		return ErrConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rev := existing.Revision + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE links SET slug=?,title=?,url=?,description=?,icon_provider=?,icon_name=?,icon_color=?,category_id=?,owner_group_id=?,enabled=?,revision=?,updated_by=?,updated_at=? WHERE id=? AND revision=?`, input.Slug, input.Title, input.URL, input.Description, input.IconProvider, input.IconName, input.IconColor, input.CategoryID, input.OwnerGroupID, btoi(input.Enabled), rev, id.UserID, now, linkID, input.Revision)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if err = s.replaceRelations(ctx, tx, linkID, input); err != nil {
		return err
	}
	if err = s.writeRevision(ctx, tx, linkID, rev, "update", snapshot{Input: input}, id.UserID); err != nil {
		return err
	}
	if err = s.audit(ctx, tx, id.UserID, "link.update", "link", fmt.Sprint(linkID), existing, input, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteLink(ctx context.Context, id policy.Identity, linkID, revision int64, requestID string) error {
	existing, err := s.q.GetLink(ctx, linkID)
	if err != nil {
		return err
	}
	p := policy.Link{ID: linkID, OwnerGroupID: existing.OwnerGroupID, CategoryID: existing.CategoryID, Enabled: existing.Enabled == 1, Deleted: existing.DeletedAt.Valid}
	if !policy.CanManage(id, p) {
		return ErrForbidden
	}
	if revision != existing.Revision {
		return ErrConflict
	}
	input, err := s.inputFor(ctx, existing)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rev := revision + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, `UPDATE links SET deleted_at=?,revision=?,updated_by=?,updated_at=? WHERE id=? AND revision=? AND deleted_at IS NULL`, now, rev, id.UserID, now, linkID, revision)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if err = s.writeRevision(ctx, tx, linkID, rev, "delete", snapshot{Input: input, Deleted: true}, id.UserID); err != nil {
		return err
	}
	if err = s.audit(ctx, tx, id.UserID, "link.delete", "link", fmt.Sprint(linkID), existing, nil, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Revisions(ctx context.Context, id policy.Identity, linkID int64) ([]RevisionView, error) {
	existing, err := s.q.GetLink(ctx, linkID)
	if err != nil {
		return nil, err
	}
	if !policy.CanManage(id, policy.Link{ID: linkID, OwnerGroupID: existing.OwnerGroupID, CategoryID: existing.CategoryID, Enabled: existing.Enabled == 1, Deleted: existing.DeletedAt.Valid}) {
		return nil, ErrForbidden
	}
	rows, err := s.q.ListRevisions(ctx, linkID)
	if err != nil {
		return nil, err
	}
	out := make([]RevisionView, 0, len(rows))
	for _, r := range rows {
		var snap any
		if err := json.Unmarshal([]byte(r.SnapshotJson), &snap); err != nil {
			return nil, err
		}
		out = append(out, RevisionView{Revision: r.Revision, Action: r.Action, Snapshot: snap, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func (s *Store) LinkHistory(ctx context.Context, id policy.Identity) ([]LinkHistoryView, error) {
	if !id.SystemAdmin && len(id.AdminOwnerGroups) == 0 && len(id.AdminCategories) == 0 {
		return nil, ErrForbidden
	}
	rows, err := s.db.QueryContext(ctx, `SELECT
		lr.link_id, l.slug, l.title, l.revision, l.deleted_at IS NOT NULL,
		l.owner_group_id, l.category_id, l.enabled,
		lr.revision, lr.action, lr.snapshot_json,
		COALESCE(u.display_name, ''), COALESCE(u.email, ''), lr.created_at
		FROM link_revisions lr
		JOIN links l ON l.id = lr.link_id
		LEFT JOIN users u ON u.id = lr.actor_user_id
		ORDER BY lr.created_at DESC, lr.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LinkHistoryView{}
	for rows.Next() {
		var item LinkHistoryView
		var ownerGroupID, categoryID, enabled, deleted int64
		var raw string
		if err := rows.Scan(&item.LinkID, &item.LinkSlug, &item.LinkTitle, &item.CurrentRevision, &deleted,
			&ownerGroupID, &categoryID, &enabled, &item.Revision, &item.Action, &raw,
			&item.ActorName, &item.ActorEmail, &item.CreatedAt); err != nil {
			return nil, err
		}
		if !policy.CanManage(id, policy.Link{ID: item.LinkID, OwnerGroupID: ownerGroupID, CategoryID: categoryID, Enabled: enabled == 1, Deleted: deleted == 1}) {
			continue
		}
		if err := json.Unmarshal([]byte(raw), &item.Snapshot); err != nil {
			return nil, err
		}
		item.CurrentDeleted = deleted == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RestoreMany(ctx context.Context, id policy.Identity, targets []RestoreTarget, requestID string) error {
	if len(targets) == 0 || len(targets) > 100 {
		return errors.New("select between 1 and 100 link revisions")
	}
	seen := make(map[int64]bool, len(targets))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.q.WithTx(tx)
	for _, target := range targets {
		if target.LinkID < 1 || target.Revision < 1 || target.CurrentRevision < 1 || seen[target.LinkID] {
			return errors.New("each selected link must have one valid revision")
		}
		seen[target.LinkID] = true
		existing, err := q.GetLink(ctx, target.LinkID)
		if err != nil {
			return err
		}
		if existing.Revision != target.CurrentRevision {
			return ErrConflict
		}
		if !policy.CanManage(id, policy.Link{ID: target.LinkID, OwnerGroupID: existing.OwnerGroupID, CategoryID: existing.CategoryID, Enabled: existing.Enabled == 1, Deleted: existing.DeletedAt.Valid}) {
			return ErrForbidden
		}
		var raw string
		if err := tx.QueryRowContext(ctx, `SELECT snapshot_json FROM link_revisions WHERE link_id=? AND revision=?`, target.LinkID, target.Revision).Scan(&raw); err != nil {
			return err
		}
		var snap snapshot
		if err := json.Unmarshal([]byte(raw), &snap); err != nil {
			return err
		}
		if err := validateInput(snap.Input); err != nil {
			return err
		}
		revision := existing.Revision + 1
		now := time.Now().UTC().Format(time.RFC3339Nano)
		var deletedAt any
		if snap.Deleted {
			deletedAt = now
		}
		result, err := tx.ExecContext(ctx, `UPDATE links SET slug=?,title=?,url=?,description=?,icon_provider=?,icon_name=?,icon_color=?,category_id=?,owner_group_id=?,enabled=?,revision=?,deleted_at=?,updated_by=?,updated_at=? WHERE id=? AND revision=?`, snap.Input.Slug, snap.Input.Title, snap.Input.URL, snap.Input.Description, snap.Input.IconProvider, snap.Input.IconName, snap.Input.IconColor, snap.Input.CategoryID, snap.Input.OwnerGroupID, btoi(snap.Input.Enabled), revision, deletedAt, id.UserID, now, target.LinkID, target.CurrentRevision)
		if err != nil {
			return err
		}
		updated, _ := result.RowsAffected()
		if updated != 1 {
			return ErrConflict
		}
		if err := s.replaceRelations(ctx, tx, target.LinkID, snap.Input); err != nil {
			return err
		}
		snap.Input.Revision = revision
		if err := s.writeRevision(ctx, tx, target.LinkID, revision, "restore", snap, id.UserID); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, id.UserID, "link.restore", "link", fmt.Sprint(target.LinkID), existing, snap, requestID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Restore(ctx context.Context, id policy.Identity, linkID, targetRevision int64, requestID string) error {
	existing, err := s.q.GetLink(ctx, linkID)
	if err != nil {
		return err
	}
	if !policy.CanManage(id, policy.Link{ID: linkID, OwnerGroupID: existing.OwnerGroupID, CategoryID: existing.CategoryID, Enabled: existing.Enabled == 1, Deleted: existing.DeletedAt.Valid}) {
		return ErrForbidden
	}
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT snapshot_json FROM link_revisions WHERE link_id=? AND revision=?`, linkID, targetRevision).Scan(&raw); err != nil {
		return err
	}
	var snap snapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return err
	}
	if err := validateInput(snap.Input); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rev := existing.Revision + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var deleted any
	if snap.Deleted {
		deleted = now
	}
	_, err = tx.ExecContext(ctx, `UPDATE links SET slug=?,title=?,url=?,description=?,icon_provider=?,icon_name=?,icon_color=?,category_id=?,owner_group_id=?,enabled=?,revision=?,deleted_at=?,updated_by=?,updated_at=? WHERE id=?`, snap.Input.Slug, snap.Input.Title, snap.Input.URL, snap.Input.Description, snap.Input.IconProvider, snap.Input.IconName, snap.Input.IconColor, snap.Input.CategoryID, snap.Input.OwnerGroupID, btoi(snap.Input.Enabled), rev, deleted, id.UserID, now, linkID)
	if err != nil {
		return err
	}
	if err = s.replaceRelations(ctx, tx, linkID, snap.Input); err != nil {
		return err
	}
	snap.Input.Revision = rev
	if err = s.writeRevision(ctx, tx, linkID, rev, "restore", snap, id.UserID); err != nil {
		return err
	}
	if err = s.audit(ctx, tx, id.UserID, "link.restore", "link", fmt.Sprint(linkID), existing, snap, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ImportSeedIfEmpty(ctx context.Context, d seed.Document) error {
	lc, e := s.q.CountLinks(ctx)
	if e != nil {
		return e
	}
	cc, e := s.q.CountCategories(ctx)
	if e != nil {
		return e
	}
	gc, e := s.q.CountOwnerGroups(ctx)
	if e != nil {
		return e
	}
	if lc+cc+gc > 0 {
		return nil
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	cats := map[string]int64{}
	for _, c := range d.Categories {
		res, e := tx.ExecContext(ctx, `INSERT INTO categories(slug,name,description,color,sort_order,created_at,updated_at)VALUES(?,?,?,?,?,?,?)`, c.Slug, c.Name, c.Description, defaultString(c.Color, "#64748b"), c.Order, now, now)
		if e != nil {
			return e
		}
		cats[c.Slug], _ = res.LastInsertId()
	}
	groups := map[string]int64{}
	for _, g := range d.Groups {
		res, e := tx.ExecContext(ctx, `INSERT INTO owner_groups(slug,name,description,created_at,updated_at)VALUES(?,?,?,?,?)`, g.Slug, g.Name, g.Description, now, now)
		if e != nil {
			return e
		}
		groups[g.Slug], _ = res.LastInsertId()
	}
	for _, l := range d.Links {
		input := LinkInput{Slug: l.Slug, Title: l.Title, URL: l.URL, Description: l.Description, IconProvider: l.Icon.Provider, IconName: l.Icon.Name, IconColor: l.Icon.Color, CategoryID: cats[l.Category], OwnerGroupID: groups[l.OwnerGroup], Tags: l.Tags, Roles: l.Audience.Roles, Teams: l.Audience.Teams, Enabled: true}
		res, e := tx.ExecContext(ctx, `INSERT INTO links(slug,title,url,description,icon_provider,icon_name,icon_color,category_id,owner_group_id,enabled,revision,created_at,updated_at)VALUES(?,?,?,?,?,?,?,?,?,1,1,?,?)`, input.Slug, input.Title, input.URL, input.Description, input.IconProvider, input.IconName, input.IconColor, input.CategoryID, input.OwnerGroupID, now, now)
		if e != nil {
			return e
		}
		linkID, _ := res.LastInsertId()
		if e = s.replaceRelations(ctx, tx, linkID, input); e != nil {
			return e
		}
		if e = s.writeRevision(ctx, tx, linkID, 1, "create", snapshot{Input: input}, 0); e != nil {
			return e
		}
	}
	return tx.Commit()
}

// ImportSeed merges a validated seed document into an existing catalog. Slugs
// are stable identities: categories and owner groups are upserted, links are
// created or updated, and soft-deleted links are restored. Catalog records that
// are absent from d are left untouched.
func (s *Store) ImportSeed(ctx context.Context, d seed.Document, requestID string) (ImportResult, error) {
	return s.ImportSeedAs(ctx, d, requestID, 0)
}

// ImportSeedAs performs a merge import and attributes its link history to an
// authenticated user. An actorID of zero represents a CLI or startup import.
func (s *Store) ImportSeedAs(ctx context.Context, d seed.Document, requestID string, actorID int64) (ImportResult, error) {
	var result ImportResult
	if err := d.Validate(); err != nil {
		return result, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	categories := make(map[string]int64, len(d.Categories))
	for _, category := range d.Categories {
		var id int64
		err = tx.QueryRowContext(ctx, `INSERT INTO categories(slug,name,description,color,sort_order,enabled,created_at,updated_at)
			VALUES(?,?,?,?,?,1,?,?)
			ON CONFLICT(slug) DO UPDATE SET name=excluded.name,description=excluded.description,color=excluded.color,sort_order=excluded.sort_order,enabled=1,updated_at=excluded.updated_at
			RETURNING id`, category.Slug, category.Name, category.Description, defaultString(category.Color, "#64748b"), category.Order, now, now).Scan(&id)
		if err != nil {
			return result, fmt.Errorf("import category %q: %w", category.Slug, err)
		}
		categories[category.Slug] = id
		result.Categories++
	}

	groups := make(map[string]int64, len(d.Groups))
	for _, group := range d.Groups {
		var id int64
		err = tx.QueryRowContext(ctx, `INSERT INTO owner_groups(slug,name,description,created_at,updated_at)
			VALUES(?,?,?,?,?)
			ON CONFLICT(slug) DO UPDATE SET name=excluded.name,description=excluded.description,updated_at=excluded.updated_at
			RETURNING id`, group.Slug, group.Name, group.Description, now, now).Scan(&id)
		if err != nil {
			return result, fmt.Errorf("import owner group %q: %w", group.Slug, err)
		}
		groups[group.Slug] = id
		result.Groups++
	}

	for _, link := range d.Links {
		input := LinkInput{
			Slug: link.Slug, Title: link.Title, URL: link.URL, Description: link.Description,
			IconProvider: link.Icon.Provider, IconName: link.Icon.Name, IconColor: link.Icon.Color,
			CategoryID: categories[link.Category], OwnerGroupID: groups[link.OwnerGroup],
			Tags: canonicalRelation(link.Tags), Roles: canonicalRelation(link.Audience.Roles),
			Teams: canonicalRelation(link.Audience.Teams), Enabled: true,
		}
		if err = validateInput(input); err != nil {
			return result, fmt.Errorf("import link %q: %w", link.Slug, err)
		}

		var id, revision int64
		var deletedAt sql.NullString
		err = tx.QueryRowContext(ctx, `SELECT id,revision,deleted_at FROM links WHERE slug=?`, link.Slug).Scan(&id, &revision, &deletedAt)
		if errors.Is(err, sql.ErrNoRows) {
			var insert sql.Result
			insert, err = tx.ExecContext(ctx, `INSERT INTO links(slug,title,url,description,icon_provider,icon_name,icon_color,category_id,owner_group_id,enabled,revision,created_at,updated_at)
				VALUES(?,?,?,?,?,?,?,?,?,1,1,?,?)`, input.Slug, input.Title, input.URL, input.Description, input.IconProvider, input.IconName, input.IconColor, input.CategoryID, input.OwnerGroupID, now, now)
			if err != nil {
				return result, fmt.Errorf("create imported link %q: %w", link.Slug, err)
			}
			id, _ = insert.LastInsertId()
			if err = s.replaceRelations(ctx, tx, id, input); err != nil {
				return result, err
			}
			input.Revision = 1
			after := snapshot{Input: input}
			if err = s.writeRevision(ctx, tx, id, 1, "create", after, actorID); err != nil {
				return result, err
			}
			if err = s.audit(ctx, tx, actorID, "link.import_create", "link", fmt.Sprint(id), nil, after, requestID); err != nil {
				return result, err
			}
			result.Created++
			continue
		}
		if err != nil {
			return result, fmt.Errorf("find imported link %q: %w", link.Slug, err)
		}

		current, err := inputForTx(ctx, tx, id)
		if err != nil {
			return result, fmt.Errorf("read imported link %q: %w", link.Slug, err)
		}
		if !deletedAt.Valid && equivalentLinkInput(current, input) {
			result.Unchanged++
			continue
		}
		action, auditAction := "update", "link.import_update"
		if deletedAt.Valid {
			action, auditAction = "restore", "link.import_restore"
			result.Restored++
		} else {
			result.Updated++
		}
		nextRevision := revision + 1
		_, err = tx.ExecContext(ctx, `UPDATE links SET title=?,url=?,description=?,icon_provider=?,icon_name=?,icon_color=?,category_id=?,owner_group_id=?,enabled=1,revision=?,deleted_at=NULL,updated_by=NULL,updated_at=? WHERE id=?`, input.Title, input.URL, input.Description, input.IconProvider, input.IconName, input.IconColor, input.CategoryID, input.OwnerGroupID, nextRevision, now, id)
		if err != nil {
			return result, fmt.Errorf("update imported link %q: %w", link.Slug, err)
		}
		if err = s.replaceRelations(ctx, tx, id, input); err != nil {
			return result, err
		}
		before := snapshot{Input: current, Deleted: deletedAt.Valid}
		input.Revision = nextRevision
		after := snapshot{Input: input}
		if err = s.writeRevision(ctx, tx, id, nextRevision, action, after, actorID); err != nil {
			return result, err
		}
		if err = s.audit(ctx, tx, actorID, auditAction, "link", fmt.Sprint(id), before, after, requestID); err != nil {
			return result, err
		}
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func inputForTx(ctx context.Context, tx *sql.Tx, linkID int64) (LinkInput, error) {
	var input LinkInput
	var enabled int64
	err := tx.QueryRowContext(ctx, `SELECT slug,title,url,description,icon_provider,icon_name,icon_color,category_id,owner_group_id,enabled,revision FROM links WHERE id=?`, linkID).Scan(
		&input.Slug, &input.Title, &input.URL, &input.Description, &input.IconProvider, &input.IconName, &input.IconColor, &input.CategoryID, &input.OwnerGroupID, &enabled, &input.Revision,
	)
	if err != nil {
		return LinkInput{}, err
	}
	input.Enabled = enabled == 1
	for query, target := range map[string]*[]string{
		`SELECT t.slug FROM tags t JOIN link_tags lt ON lt.tag_id=t.id WHERE lt.link_id=? ORDER BY t.slug`:    &input.Tags,
		`SELECT r.slug FROM roles r JOIN link_roles lr ON lr.role_id=r.id WHERE lr.link_id=? ORDER BY r.slug`: &input.Roles,
		`SELECT t.slug FROM teams t JOIN link_teams lt ON lt.team_id=t.id WHERE lt.link_id=? ORDER BY t.slug`: &input.Teams,
	} {
		rows, queryErr := tx.QueryContext(ctx, query, linkID)
		if queryErr != nil {
			return LinkInput{}, queryErr
		}
		for rows.Next() {
			var value string
			if queryErr = rows.Scan(&value); queryErr != nil {
				rows.Close()
				return LinkInput{}, queryErr
			}
			*target = append(*target, value)
		}
		queryErr = rows.Err()
		rows.Close()
		if queryErr != nil {
			return LinkInput{}, queryErr
		}
	}
	return input, nil
}

func canonicalRelation(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range unique(values) {
		if slug := normalizeSlug(value); slug != "" {
			result = append(result, slug)
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func equivalentLinkInput(a, b LinkInput) bool {
	a.Revision, b.Revision = 0, 0
	a.Tags, a.Roles, a.Teams = canonicalRelation(a.Tags), canonicalRelation(a.Roles), canonicalRelation(a.Teams)
	b.Tags, b.Roles, b.Teams = canonicalRelation(b.Tags), canonicalRelation(b.Roles), canonicalRelation(b.Teams)
	return a.Slug == b.Slug && a.Title == b.Title && a.URL == b.URL && a.Description == b.Description &&
		a.IconProvider == b.IconProvider && a.IconName == b.IconName && a.IconColor == b.IconColor &&
		a.CategoryID == b.CategoryID && a.OwnerGroupID == b.OwnerGroupID && a.Enabled == b.Enabled &&
		slices.Equal(a.Tags, b.Tags) && slices.Equal(a.Roles, b.Roles) && slices.Equal(a.Teams, b.Teams)
}

var ErrForbidden = errors.New("forbidden")
var ErrConflict = errors.New("revision conflict")

func (s *Store) replaceRelations(ctx context.Context, tx *sql.Tx, linkID int64, input LinkInput) error {
	for _, table := range []string{"link_tags", "link_roles", "link_teams"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE link_id=?", linkID); err != nil {
			return err
		}
	}
	for _, v := range unique(input.Tags) {
		slug := normalizeSlug(v)
		if slug == "" {
			continue
		}
		var id int64
		if err := tx.QueryRowContext(ctx, `INSERT INTO tags(slug,name)VALUES(?,?)ON CONFLICT(slug)DO UPDATE SET name=excluded.name RETURNING id`, slug, v).Scan(&id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO link_tags(link_id,tag_id)VALUES(?,?)`, linkID, id); err != nil {
			return err
		}
	}
	for _, x := range []struct {
		values           []string
		entity, join, fk string
	}{{input.Roles, "roles", "link_roles", "role_id"}, {input.Teams, "teams", "link_teams", "team_id"}} {
		for _, v := range unique(x.values) {
			slug := normalizeSlug(v)
			if slug == "" {
				continue
			}
			var id int64
			q := fmt.Sprintf(`INSERT INTO %s(slug,name)VALUES(?,?)ON CONFLICT(slug)DO UPDATE SET name=excluded.name RETURNING id`, x.entity)
			if err := tx.QueryRowContext(ctx, q, slug, v).Scan(&id); err != nil {
				return err
			}
			q = fmt.Sprintf(`INSERT INTO %s(link_id,%s)VALUES(?,?)`, x.join, x.fk)
			if _, err := tx.ExecContext(ctx, q, linkID, id); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Store) writeRevision(ctx context.Context, tx *sql.Tx, linkID, revision int64, action string, snap snapshot, actor int64) error {
	snap.Input.Revision = revision
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	var a any
	if actor > 0 {
		a = actor
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO link_revisions(link_id,revision,action,snapshot_json,actor_user_id,created_at)VALUES(?,?,?,?,?,?)`, linkID, revision, action, string(b), a, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) audit(ctx context.Context, tx *sql.Tx, actor int64, action, targetType, targetID string, before, after any, requestID string) error {
	var a any
	if actor > 0 {
		a = actor
	}
	bj, _ := json.Marshal(before)
	aj, _ := json.Marshal(after)
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_user_id,action,target_type,target_id,before_json,after_json,request_id,created_at)VALUES(?,?,?,?,?,?,?,?)`, a, action, targetType, targetID, nullJSON(bj), nullJSON(aj), requestID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) inputFor(ctx context.Context, r dbgen.GetLinkRow) (LinkInput, error) {
	tags, e := s.q.ListLinkTags(ctx, r.ID)
	if e != nil {
		return LinkInput{}, e
	}
	roles, e := s.q.ListLinkRoles(ctx, r.ID)
	if e != nil {
		return LinkInput{}, e
	}
	teams, e := s.q.ListLinkTeams(ctx, r.ID)
	if e != nil {
		return LinkInput{}, e
	}
	return LinkInput{Slug: r.Slug, Title: r.Title, URL: r.Url, Description: r.Description, IconProvider: r.IconProvider, IconName: r.IconName, IconColor: r.IconColor, CategoryID: r.CategoryID, OwnerGroupID: r.OwnerGroupID, Tags: tags, Roles: roles, Teams: teams, Enabled: r.Enabled == 1, Revision: r.Revision}, nil
}
func validateInput(i LinkInput) error {
	if !slugRE.MatchString(i.Slug) {
		return errors.New("invalid slug")
	}
	if strings.TrimSpace(i.Title) == "" {
		return errors.New("title is required")
	}
	if len(i.URL) > 2048 {
		return errors.New("URL is too long")
	}
	u, e := url.Parse(i.URL)
	if e != nil || u.Scheme == "" {
		return errors.New("valid URI with a scheme is required")
	}
	switch strings.ToLower(u.Scheme) {
	case "javascript", "data", "vbscript":
		return errors.New("blocked URI scheme")
	}
	if u.User != nil {
		return errors.New("embedded URI credentials are not allowed")
	}
	if i.CategoryID < 1 || i.OwnerGroupID < 1 {
		return errors.New("category and owner group are required")
	}
	return nil
}
func unique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
func normalizeSlug(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(v, "-")
	return strings.Trim(v, "-")
}
func prefsView(p dbgen.UserPreference) Preferences {
	return Preferences{Theme: p.Theme, ViewMode: p.ViewMode, ShowDescriptions: p.ShowDescriptions == 1, ShowCategories: p.ShowCategories == 1, ShowIcons: p.ShowIcons == 1, ColoredIcons: p.ColoredIcons == 1, ShowHidden: p.ShowHidden == 1, OpenNewTab: p.OpenNewTab == 1, HotkeysEnabled: p.HotkeysEnabled == 1, Revision: p.Revision}
}
func btoi(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
func defaultString(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func nullJSON(b []byte) any {
	if string(b) == "null" {
		return nil
	}
	return string(b)
}
