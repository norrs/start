-- name: UpsertUser :one
INSERT INTO users (issuer, subject, email, display_name, claims_refreshed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (issuer, subject) DO UPDATE SET
  email = excluded.email,
  display_name = excluded.display_name,
  claims_refreshed_at = excluded.claims_refreshed_at,
  updated_at = excluded.updated_at
RETURNING *;

-- name: GetUserByIdentity :one
SELECT * FROM users WHERE issuer = ? AND subject = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: ReplaceUserOIDCGroupsDelete :exec
DELETE FROM user_oidc_groups WHERE user_id = ?;

-- name: AddUserOIDCGroup :exec
INSERT INTO user_oidc_groups (user_id, group_value, claim_seen_at) VALUES (?, ?, ?);

-- name: ListUserOIDCGroups :many
SELECT group_value FROM user_oidc_groups WHERE user_id = ? ORDER BY group_value;

-- name: ListCategories :many
SELECT * FROM categories WHERE enabled = 1 ORDER BY sort_order, name;

-- name: ListOwnerGroups :many
SELECT * FROM owner_groups ORDER BY name;

-- name: ListActiveLinks :many
SELECT l.*, c.slug AS category_slug, c.name AS category_name, c.color AS category_color,
       g.slug AS owner_group_slug, g.name AS owner_group_name
FROM links l
JOIN categories c ON c.id = l.category_id
JOIN owner_groups g ON g.id = l.owner_group_id
WHERE l.deleted_at IS NULL AND l.enabled = 1 AND c.enabled = 1
ORDER BY c.sort_order, c.name, l.title;

-- name: GetLink :one
SELECT l.*, c.slug AS category_slug, c.name AS category_name, c.color AS category_color,
       g.slug AS owner_group_slug, g.name AS owner_group_name
FROM links l
JOIN categories c ON c.id = l.category_id
JOIN owner_groups g ON g.id = l.owner_group_id
WHERE l.id = ?;

-- name: ListLinkTags :many
SELECT t.slug FROM tags t JOIN link_tags lt ON lt.tag_id = t.id WHERE lt.link_id = ? ORDER BY t.slug;

-- name: ListLinkRoles :many
SELECT r.slug FROM roles r JOIN link_roles lr ON lr.role_id = r.id WHERE lr.link_id = ? ORDER BY r.slug;

-- name: ListLinkTeams :many
SELECT t.slug FROM teams t JOIN link_teams lt ON lt.team_id = t.id WHERE lt.link_id = ? ORDER BY t.slug;

-- name: ListUserRoleSlugs :many
SELECT DISTINCT r.slug FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = ? ORDER BY r.slug;

-- name: ListUserTeamSlugs :many
SELECT DISTINCT t.slug FROM teams t JOIN user_teams ut ON ut.team_id = t.id WHERE ut.user_id = ? ORDER BY t.slug;

-- name: GetPreferences :one
SELECT * FROM user_preferences WHERE user_id = ?;

-- name: UpsertPreferences :one
INSERT INTO user_preferences (
  user_id, theme, view_mode, show_descriptions, show_categories, show_icons,
  colored_icons, show_hidden, open_new_tab, hotkeys_enabled, revision, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
ON CONFLICT (user_id) DO UPDATE SET
  theme = excluded.theme,
  view_mode = excluded.view_mode,
  show_descriptions = excluded.show_descriptions,
  show_categories = excluded.show_categories,
  show_icons = excluded.show_icons,
  colored_icons = excluded.colored_icons,
  show_hidden = excluded.show_hidden,
  open_new_tab = excluded.open_new_tab,
  hotkeys_enabled = excluded.hotkeys_enabled,
  revision = user_preferences.revision + 1,
  updated_at = excluded.updated_at
RETURNING *;

-- name: ListHiddenLinkIDs :many
SELECT link_id FROM user_hidden_links WHERE user_id = ? ORDER BY link_id;

-- name: HideLink :exec
INSERT INTO user_hidden_links (user_id, link_id, hidden_at) VALUES (?, ?, ?)
ON CONFLICT (user_id, link_id) DO NOTHING;

-- name: ShowLink :exec
DELETE FROM user_hidden_links WHERE user_id = ? AND link_id = ?;

-- name: GetAppSettings :one
SELECT * FROM app_settings WHERE id = 1;

-- name: CountLinks :one
SELECT COUNT(*) FROM links;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories;

-- name: CountOwnerGroups :one
SELECT COUNT(*) FROM owner_groups;

-- name: InsertCategory :one
INSERT INTO categories (slug, name, description, color, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: InsertOwnerGroup :one
INSERT INTO owner_groups (slug, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: InsertRole :one
INSERT INTO roles (slug, name) VALUES (?, ?) ON CONFLICT (slug) DO UPDATE SET name = excluded.name RETURNING *;

-- name: InsertTeam :one
INSERT INTO teams (slug, name) VALUES (?, ?) ON CONFLICT (slug) DO UPDATE SET name = excluded.name RETURNING *;

-- name: ListRevisions :many
SELECT * FROM link_revisions WHERE link_id = ? ORDER BY revision DESC;
