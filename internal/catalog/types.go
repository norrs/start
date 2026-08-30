package catalog

type UserView struct {
	ID             int64    `json:"id"`
	Email          string   `json:"email"`
	DisplayName    string   `json:"displayName"`
	Roles          []string `json:"roles"`
	Teams          []string `json:"teams"`
	Groups         []string `json:"groups"`
	SystemAdmin    bool     `json:"systemAdmin"`
	CanManageLinks bool     `json:"canManageLinks"`
}

type Branding struct {
	AppName     string `json:"appName"`
	WelcomeText string `json:"welcomeText"`
	LogoURL     string `json:"logoUrl"`
	AccentColor string `json:"accentColor"`
	SupportURL  string `json:"supportUrl"`
}

type Preferences struct {
	Theme            string `json:"theme"`
	ViewMode         string `json:"viewMode"`
	ShowDescriptions bool   `json:"showDescriptions"`
	ShowCategories   bool   `json:"showCategories"`
	ShowIcons        bool   `json:"showIcons"`
	ColoredIcons     bool   `json:"coloredIcons"`
	ShowHidden       bool   `json:"showHidden"`
	OpenNewTab       bool   `json:"openNewTab"`
	HotkeysEnabled   bool   `json:"hotkeysEnabled"`
	Revision         int64  `json:"revision"`
}

type CategoryView struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	SortOrder   int64  `json:"sortOrder"`
}

type OwnerGroupView struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CanCreate   bool   `json:"canCreate"`
}

type LinkView struct {
	ID             int64    `json:"id"`
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	URL            string   `json:"url"`
	Description    string   `json:"description"`
	IconProvider   string   `json:"iconProvider"`
	IconName       string   `json:"iconName"`
	IconColor      string   `json:"iconColor"`
	CategoryID     int64    `json:"categoryId"`
	CategorySlug   string   `json:"categorySlug"`
	CategoryName   string   `json:"categoryName"`
	CategoryColor  string   `json:"categoryColor"`
	OwnerGroupID   int64    `json:"ownerGroupId"`
	OwnerGroupSlug string   `json:"ownerGroupSlug"`
	OwnerGroupName string   `json:"ownerGroupName"`
	Tags           []string `json:"tags"`
	Roles          []string `json:"roles"`
	Teams          []string `json:"teams"`
	Hidden         bool     `json:"hidden"`
	CanManage      bool     `json:"canManage"`
	Revision       int64    `json:"revision"`
}

type Bootstrap struct {
	User        UserView         `json:"user"`
	Branding    Branding         `json:"branding"`
	Preferences Preferences      `json:"preferences"`
	Categories  []CategoryView   `json:"categories"`
	OwnerGroups []OwnerGroupView `json:"ownerGroups"`
	Links       []LinkView       `json:"links"`
}

type LinkInput struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	Description  string   `json:"description"`
	IconProvider string   `json:"iconProvider"`
	IconName     string   `json:"iconName"`
	IconColor    string   `json:"iconColor"`
	CategoryID   int64    `json:"categoryId"`
	OwnerGroupID int64    `json:"ownerGroupId"`
	Tags         []string `json:"tags"`
	Roles        []string `json:"roles"`
	Teams        []string `json:"teams"`
	Enabled      bool     `json:"enabled"`
	Revision     int64    `json:"revision"`
}

// ImportResult summarizes a merge import. Records not present in the imported
// document are never changed.
type ImportResult struct {
	Categories int `json:"categories"`
	Groups     int `json:"groups"`
	Created    int `json:"created"`
	Updated    int `json:"updated"`
	Restored   int `json:"restored"`
	Unchanged  int `json:"unchanged"`
}

type RevisionView struct {
	Revision  int64  `json:"revision"`
	Action    string `json:"action"`
	Snapshot  any    `json:"snapshot"`
	CreatedAt string `json:"createdAt"`
}

type LinkHistoryView struct {
	LinkID          int64  `json:"linkId"`
	LinkSlug        string `json:"linkSlug"`
	LinkTitle       string `json:"linkTitle"`
	CurrentRevision int64  `json:"currentRevision"`
	CurrentDeleted  bool   `json:"currentDeleted"`
	Revision        int64  `json:"revision"`
	Action          string `json:"action"`
	Snapshot        any    `json:"snapshot"`
	ActorName       string `json:"actorName"`
	ActorEmail      string `json:"actorEmail"`
	CreatedAt       string `json:"createdAt"`
}

type RestoreTarget struct {
	LinkID          int64 `json:"linkId"`
	Revision        int64 `json:"revision"`
	CurrentRevision int64 `json:"currentRevision"`
}

type snapshot struct {
	Input   LinkInput `json:"input"`
	Deleted bool      `json:"deleted"`
}
