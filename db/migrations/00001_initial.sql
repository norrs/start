-- +goose Up
PRAGMA foreign_keys = ON;

CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issuer TEXT NOT NULL,
  subject TEXT NOT NULL,
  email TEXT NOT NULL,
  display_name TEXT NOT NULL,
  claims_refreshed_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (issuer, subject)
);

CREATE TABLE sessions (
  token TEXT PRIMARY KEY,
  data BLOB NOT NULL,
  expiry TEXT NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);

CREATE TABLE roles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL
);

CREATE TABLE teams (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL
);

CREATE TABLE user_oidc_groups (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  group_value TEXT NOT NULL,
  claim_seen_at TEXT NOT NULL,
  PRIMARY KEY (user_id, group_value)
);

CREATE TABLE identity_group_mappings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_value TEXT NOT NULL,
  target_type TEXT NOT NULL CHECK (target_type IN ('role', 'team')),
  target_id INTEGER NOT NULL,
  UNIQUE (group_value, target_type, target_id)
);

CREATE TABLE user_roles (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  source TEXT NOT NULL CHECK (source IN ('local', 'oidc')),
  last_seen_at TEXT NOT NULL,
  PRIMARY KEY (user_id, role_id, source)
);

CREATE TABLE user_teams (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  source TEXT NOT NULL CHECK (source IN ('local', 'oidc')),
  last_seen_at TEXT NOT NULL,
  PRIMARY KEY (user_id, team_id, source)
);

CREATE TABLE owner_groups (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL DEFAULT '#64748b',
  sort_order INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE admin_grants (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_type TEXT NOT NULL CHECK (source_type IN ('user', 'oidc_group')),
  user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
  oidc_group_value TEXT,
  capability TEXT NOT NULL CHECK (capability IN ('system_admin', 'group_admin', 'category_admin')),
  owner_group_id INTEGER REFERENCES owner_groups(id) ON DELETE CASCADE,
  category_id INTEGER REFERENCES categories(id) ON DELETE CASCADE,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL,
  CHECK ((source_type = 'user' AND user_id IS NOT NULL AND oidc_group_value IS NULL)
      OR (source_type = 'oidc_group' AND user_id IS NULL AND oidc_group_value IS NOT NULL)),
  CHECK ((capability = 'system_admin' AND owner_group_id IS NULL AND category_id IS NULL)
      OR (capability = 'group_admin' AND owner_group_id IS NOT NULL AND category_id IS NULL)
      OR (capability = 'category_admin' AND owner_group_id IS NULL AND category_id IS NOT NULL))
);
CREATE UNIQUE INDEX admin_grants_unique_user ON admin_grants(source_type, user_id, capability, owner_group_id, category_id) WHERE source_type = 'user';
CREATE UNIQUE INDEX admin_grants_unique_group ON admin_grants(source_type, oidc_group_value, capability, owner_group_id, category_id) WHERE source_type = 'oidc_group';

CREATE TABLE links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  url TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  icon_provider TEXT NOT NULL DEFAULT '',
  icon_name TEXT NOT NULL DEFAULT '',
  icon_color TEXT NOT NULL DEFAULT '',
  category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
  owner_group_id INTEGER NOT NULL REFERENCES owner_groups(id) ON DELETE RESTRICT,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  revision INTEGER NOT NULL DEFAULT 1,
  deleted_at TEXT,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX links_active_category_idx ON links(category_id, deleted_at, enabled);
CREATE INDEX links_owner_group_idx ON links(owner_group_id, deleted_at);

CREATE TABLE tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL
);
CREATE TABLE link_tags (
  link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (link_id, tag_id)
);
CREATE TABLE link_roles (
  link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (link_id, role_id)
);
CREATE TABLE link_teams (
  link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
  PRIMARY KEY (link_id, team_id)
);

CREATE TABLE link_revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  revision INTEGER NOT NULL,
  action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete', 'restore')),
  snapshot_json TEXT NOT NULL,
  actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL,
  UNIQUE (link_id, revision)
);

CREATE TABLE user_hidden_links (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  link_id INTEGER NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  hidden_at TEXT NOT NULL,
  PRIMARY KEY (user_id, link_id)
);

CREATE TABLE user_preferences (
  user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  theme TEXT NOT NULL DEFAULT 'dark' CHECK (theme IN ('dark', 'light', 'hacker', '1984')),
  view_mode TEXT NOT NULL DEFAULT 'categorized' CHECK (view_mode IN ('categorized', 'flat')),
  show_descriptions INTEGER NOT NULL DEFAULT 1 CHECK (show_descriptions IN (0, 1)),
  show_categories INTEGER NOT NULL DEFAULT 1 CHECK (show_categories IN (0, 1)),
  show_icons INTEGER NOT NULL DEFAULT 1 CHECK (show_icons IN (0, 1)),
  colored_icons INTEGER NOT NULL DEFAULT 1 CHECK (colored_icons IN (0, 1)),
  show_hidden INTEGER NOT NULL DEFAULT 0 CHECK (show_hidden IN (0, 1)),
  open_new_tab INTEGER NOT NULL DEFAULT 0 CHECK (open_new_tab IN (0, 1)),
  hotkeys_enabled INTEGER NOT NULL DEFAULT 1 CHECK (hotkeys_enabled IN (0, 1)),
  revision INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
);

CREATE TABLE app_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  app_name TEXT NOT NULL DEFAULT 'Start',
  welcome_text TEXT NOT NULL DEFAULT 'Internal tools and resources',
  logo_url TEXT NOT NULL DEFAULT '',
  accent_color TEXT NOT NULL DEFAULT '#5eead4',
  support_url TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
);

CREATE TABLE audit_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  before_json TEXT,
  after_json TEXT,
  request_id TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX audit_events_created_idx ON audit_events(created_at);

INSERT INTO app_settings (id, updated_at) VALUES (1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));

-- +goose Down
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS user_hidden_links;
DROP TABLE IF EXISTS link_revisions;
DROP TABLE IF EXISTS link_teams;
DROP TABLE IF EXISTS link_roles;
DROP TABLE IF EXISTS link_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS links;
DROP TABLE IF EXISTS admin_grants;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS owner_groups;
DROP TABLE IF EXISTS user_teams;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS identity_group_mappings;
DROP TABLE IF EXISTS user_oidc_groups;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
