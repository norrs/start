export interface User { id:number; email:string; displayName:string; roles:string[]; teams:string[]; groups:string[]; systemAdmin:boolean; canManageLinks:boolean }
export interface Branding { appName:string; welcomeText:string; logoUrl:string; accentColor:string; supportUrl:string }
export interface Preferences { theme:'dark'|'light'|'hacker'|'1984'; viewMode:'categorized'|'flat'; showDescriptions:boolean; showCategories:boolean; showIcons:boolean; coloredIcons:boolean; showHidden:boolean; openNewTab:boolean; hotkeysEnabled:boolean; revision:number }
export interface Category { id:number; slug:string; name:string; description:string; color:string; sortOrder:number }
export interface OwnerGroup { id:number; slug:string; name:string; description:string; canCreate:boolean }
export interface Link { id:number; slug:string; title:string; url:string; description:string; iconProvider:string; iconName:string; iconColor:string; categoryId:number; categorySlug:string; categoryName:string; categoryColor:string; ownerGroupId:number; ownerGroupSlug:string; ownerGroupName:string; tags:string[]; roles:string[]; teams:string[]; hidden:boolean; canManage:boolean; revision:number }
export interface Bootstrap { user:User; branding:Branding; preferences:Preferences; categories:Category[]; ownerGroups:OwnerGroup[]; links:Link[] }
export interface LinkInput { slug:string; title:string; url:string; description:string; iconProvider:string; iconName:string; iconColor:string; categoryId:number; ownerGroupId:number; tags:string[]; roles:string[]; teams:string[]; enabled:boolean; revision:number }
export interface Revision { revision:number; action:'create'|'update'|'delete'|'restore'; snapshot:{ input:LinkInput; deleted:boolean }; createdAt:string }
export interface LinkHistoryEntry { linkId:number; linkSlug:string; linkTitle:string; currentRevision:number; currentDeleted:boolean; revision:number; action:'create'|'update'|'delete'|'restore'; snapshot:{ input:LinkInput; deleted:boolean }; actorName:string; actorEmail:string; createdAt:string }
export interface RestoreTarget { linkId:number; revision:number; currentRevision:number }
export interface ImportResult { categories:number; groups:number; created:number; updated:number; restored:number; unchanged:number }
