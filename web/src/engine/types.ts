export type InstanceRole = "owner" | "admin" | "member";
export type WorkspaceRole = "admin" | "member" | "viewer";
export type AccessLevel = "full" | "edit" | "content" | "create" | "comment" | "view";
export type TeamAccess = "open" | "closed" | "private";
export type AgentAccess = "open" | "strict" | "closed";

export interface Prefs {
  lang: string;
  region: string;
  time_zone: string;
  hour_cycle: "" | "h12" | "h23";
  week_start: "" | "monday" | "sunday" | "saturday";
  notes_mode: boolean;
  system_font: boolean;
}

export interface User {
  id: string;
  email: string;
  name: string;
  color: string;
  avatar: string;
  role: InstanceRole;
  email_verified: boolean;
  disabled: boolean;
  totp_enabled: boolean;
  prefs: Prefs;
  created_at: number;
}

export interface Workspace {
  id: string;
  name: string;
  icon: string;
  image: string;
  role: WorkspaceRole | "guest";
  personal: boolean;
  auto_join: boolean;
  sidebar_mode: "split" | "tree";
  agent_access: AgentAccess;
  has_rules: boolean;
  proposal_waiting: boolean;
  member_count: number;
  break_glass?: boolean;
  owner_id: string;
  rules_updated_at: number;
}

export interface Me {
  user: User;
  credential: { kind: "session" | "token" | "oauth"; scope: "read" | "write" };
  instance: { name: string };
  workspaces: Workspace[];
  withheld_workspaces: number;
  can_create_workspaces: boolean;
}

export interface SSOProvider {
  id: string;
  name: string;
}

export interface Boot {
  setup?: boolean;
  me?: Me;
  sso?: SSOProvider[];
  signup?: "open" | "closed";
  instance_name?: string;
  workspace_id?: string;
  seq?: number;
  pages?: Page[];
  favorites?: Favorite[];
  tags?: TagInfo[];
  teamspaces?: Teamspace[];
  comments?: CommentListing;
  inbox_unread?: number;
  page?: PageDetail;
  collection?: Collection;
  query?: { view_id: string; result: QueryResult };
  row?: RowDetail;
  form?: FormDef;
}

export type PropType =
  | "title" | "text" | "number" | "select" | "multi_select" | "status" | "date" | "person" | "files" | "checkbox" | "url" | "email" | "phone"
  | "relation" | "rollup" | "formula" | "created_time" | "created_by" | "last_edited_time" | "last_edited_by" | "unique_id" | "place" | "button";

export interface PropOption {
  id: string;
  name: string;
  color: string;
  group?: "todo" | "doing" | "done";
}

export interface ButtonAction {
  kind: string;
  config?: Record<string, unknown>;
}

export interface Property {
  id: string;
  name: string;
  type: PropType;
  description?: string;
  options?: PropOption[];
  format?: string;
  display?: "" | "bar" | "ring";
  precision?: number;
  max?: number;
  color?: string;
  time_format?: "" | "12" | "24";
  collection?: string;
  pair?: string;
  limit?: number;
  relation?: string;
  target?: string;
  fn?: string;
  expr?: string;
  prefix?: string;
  actions?: ButtonAction[];
}

export interface Filter {
  op?: "and" | "or";
  items?: Filter[];
  prop?: string;
  cond?: string;
  value?: unknown;
}

export interface SortKey {
  prop: string;
  dir: "asc" | "desc";
}

export interface Grouping {
  prop: string;
  by?: string;
  step?: number;
  sort?: "asc" | "desc" | "manual";
  order?: string[];
  hidden?: string[];
  hide_empty?: boolean;
}

export interface ViewProp {
  id: string;
  hidden?: boolean;
  width?: number;
}

export interface FormQuestion {
  prop: string;
  label?: string;
  help?: string;
  required?: boolean;
  when?: Filter;
}

export interface FormConfig {
  title?: string;
  description?: string;
  questions?: FormQuestion[];
  submit?: string;
  done?: string;
  anonymous?: boolean;
  closed?: boolean;
}

export interface ViewConfig {
  props?: ViewProp[];
  filter?: Filter;
  sorts?: SortKey[];
  group?: Grouping;
  sub_group?: Grouping;
  calcs?: Record<string, string>;
  layout?: Record<string, unknown>;
  form?: FormConfig;
}

export type ViewType = "table" | "board" | "list" | "gallery" | "calendar" | "timeline" | "form" | "chart" | "map" | "feed" | "dashboard";

export interface CollectionView {
  id: string;
  collection_id: string;
  sort_key: string;
  name: string;
  type: ViewType;
  config: ViewConfig;
  created_at: number;
  updated_at: number;
  form_token?: string;
}

export interface Collection {
  id: string;
  workspace_id: string;
  title: string;
  icon: string;
  inline: boolean;
  schema: Property[];
  views: CollectionView[];
  can_write: boolean;
  can_design: boolean;
  rows: number;
  epoch: number;
}

export type CellValue = unknown;

export interface ComputedValue {
  type: "number" | "text" | "boolean" | "date" | "list" | "person" | "page" | "error";
  value: unknown;
  href?: string;
  style?: string;
}

export interface RowData {
  id: string;
  title: string;
  icon: string;
  cover?: string;
  thumb?: string;
  snippet?: string;
  sort_key: string;
  values: Record<string, CellValue>;
  number: number;
  version: number;
  created_at: number;
  created_by: string;
  updated_at: number;
  updated_by: string;
}

export interface GroupResult {
  key: string;
  label: string;
  color?: string;
  count: number;
  rows: string[];
  calcs?: Record<string, ComputedValue | null>;
  hidden?: boolean;
}

export interface LaneResult {
  key: string;
  label: string;
  color?: string;
  count: number;
  groups: GroupResult[];
  hidden?: boolean;
}

export interface RowRef {
  title: string;
  icon: string;
  collection: string;
}

export interface QueryResult {
  total: number;
  rows: RowData[];
  order: string[];
  groups?: GroupResult[];
  lanes?: LaneResult[];
  calcs?: Record<string, ComputedValue | null>;
  refs: Record<string, RowRef>;
  epoch: number;
  offset: number;
}

export interface QueryInput {
  view_id?: string;
  filter?: Filter;
  sorts?: SortKey[];
  search?: string;
  offset?: number;
  limit?: number;
  group_key?: string;
  lane_key?: string;
  group_limit?: number;
  range?: { prop: string; end_prop?: string; from: string; to: string };
  ids?: string[];
  no_groups?: boolean;
}

export interface RowDetail {
  collection: Collection;
  row: RowData | null;
  refs: Record<string, RowRef>;
}

export interface SchemaOp {
  op: "add" | "update" | "delete" | "move";
  id?: string;
  after?: string;
  prop?: Partial<Property>;
  two_way?: boolean;
  pair_name?: string;
}

export interface FormQuestionDef extends FormQuestion {
  name: string;
  type: PropType;
  options?: PropOption[];
  limit?: number;
  collection?: string;
  format?: string;
}

export interface FormDef {
  view_id: string;
  collection_id: string;
  workspace_id?: string;
  title: string;
  description: string;
  icon: string;
  questions: FormQuestionDef[];
  submit: string;
  done: string;
  public: boolean;
  token?: string;
  closed: boolean;
  anonymous: boolean;
}

export interface CSVPreview {
  token: string;
  name: string;
  rows: number;
  columns: { name: string; type: PropType | "skip"; samples: string[] }[];
}

export type PageKind = "doc" | "collection";
export type Visibility = "workspace" | "private";

export interface Page {
  id: string;
  workspace_id: string;
  parent_id: string;
  kind: PageKind;
  sort_key: string;
  title: string;
  icon: string;
  cover: string;
  description: string;
  tags: string[];
  visibility: Visibility;
  owner_id: string;
  is_template: boolean;
  snippet: string;
  thumb: string;
  created_at: number;
  created_by: string;
  updated_at: number;
  updated_by: string;
  trashed_at?: number;
  trashed_by?: string;
  trashed_by_kind?: string;
  row?: boolean;
  teamspace_id?: string;
  detached?: boolean;
  access?: AccessLevel;
}

export interface Teamspace {
  id: string;
  workspace_id: string;
  name: string;
  icon: string;
  description: string;
  access: TeamAccess;
  sort_key: string;
  role: "" | "owner" | "member";
  members: number;
  can_manage: boolean;
}

export interface TeamMember {
  user_id: string;
  name: string;
  email: string;
  role: string;
}

export interface Group {
  id: string;
  workspace_id: string;
  name: string;
  external_id?: string;
  members: number;
}

export interface Share {
  id: string;
  page_id: string;
  kind: "user" | "group";
  subject_id: string;
  name: string;
  email?: string;
  level: AccessLevel;
  expires_at: number;
  guest?: boolean;
  inherited?: boolean;
  from_title?: string;
}

export interface ShareLink {
  id: string;
  page_id: string;
  level: "view" | "comment";
  password: boolean;
  expires_at: number;
  created_at: number;
  token?: string;
}

export interface GuestInvite {
  id: string;
  email: string;
  level: AccessLevel;
  expires_at: number;
}

export interface Sharing {
  general: "workspace" | "teamspace" | "private";
  teamspace?: Teamspace;
  shares: Share[];
  links: ShareLink[];
  invites: GuestInvite[];
  can_share: boolean;
}

export interface SharedPage extends Page {
  workspace_name: string;
  level: AccessLevel;
}

export type ReactionIcon = "heart" | "star" | "sparkle" | "eye" | "rocket" | "lightbulb";

export interface CommentReaction {
  icon: ReactionIcon;
  count: number;
  mine: boolean;
  names: string[];
}

export interface PageComment {
  id: string;
  thread_id: string;
  body: string;
  author_id: string;
  author_name: string;
  author_color: string;
  author_avatar: string;
  created_at: number;
  edited_at: number;
  reactions: CommentReaction[];
}

export interface CommentThread {
  id: string;
  page_id: string;
  anchor: "page" | "text" | "block";
  start: string;
  end: string;
  quote: string;
  resolved_at: number;
  resolved_by: string;
  created_by: string;
  created_at: number;
  updated_at: number;
  comments: PageComment[];
}

export interface Suggestion {
  id: string;
  page_id: string;
  start: string;
  end: string;
  original: string;
  text: string;
  author_id: string;
  author_name: string;
  author_color: string;
  author_avatar: string;
  status: "open" | "accepted" | "rejected";
  decided_by: string;
  decided_at: number;
  created_at: number;
  updated_at: number;
}

export interface CommentListing {
  threads: CommentThread[];
  suggestions: Suggestion[];
  can_comment: boolean;
  can_moderate: boolean;
  can_decide: boolean;
}

export interface PageVersion {
  id: string;
  page_id: string;
  created_at: number;
  kind: "auto" | "manual" | "restore";
  title: string;
  editors: string[];
  names: string[];
  words: number;
  size: number;
}

export interface Site {
  id: string;
  workspace_id: string;
  page_id: string;
  page_title: string;
  slug: string;
  hostname: string;
  title: string;
  description: string;
  image: string;
  theme: "auto" | "light" | "dark";
  font: "sans" | "serif" | "mono";
  password: boolean;
  indexing: boolean;
  subpages: boolean;
  created_by: string;
  created_at: number;
  updated_at: number;
  path: string;
}

export interface Webhook {
  id: string;
  workspace_id: string;
  kind: "webhook" | "siem";
  url: string;
  description: string;
  events: string[];
  format: "json" | "splunk";
  has_auth: boolean;
  active: boolean;
  failures: number;
  created_by: string;
  created_at: number;
  updated_at: number;
  last_status: "" | "delivered" | "failed";
  last_at: number;
}

export interface WebhookInput {
  kind?: Webhook["kind"];
  url?: string;
  description?: string;
  events?: string[];
  format?: Webhook["format"];
  auth_header?: string;
  active?: boolean;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  event: string;
  event_id: string;
  status: "pending" | "delivered" | "failed";
  attempts: number;
  next_at: number;
  last_status: number;
  last_error: string;
  created_at: number;
  delivered_at: number;
  payload?: string;
}

export interface ScimToken {
  id: string;
  label: string;
  created_by: string;
  created_at: number;
  last_used_at: number;
}

export interface AuditEntry {
  id: number;
  at: number;
  actor_kind: "human" | "agent" | "form" | "system";
  actor_id: string;
  actor_label: string;
  agent: string;
  action: string;
  workspace_id: string;
  page_id: string;
  page_title: string;
  detail: string;
  changes?: { field: string; before: unknown; after: unknown }[];
}

export interface AuditQuery {
  actor?: string;
  action?: string;
  page?: string;
  from?: number;
  to?: number;
  before?: number;
  limit?: number;
}

export interface AuditPage {
  entries: AuditEntry[];
  next: number;
  retention_days: number;
  actions?: string[];
  actors?: { id: string; label: string }[];
}

export interface AnalyticsPerson {
  id: string;
  name: string;
  email?: string;
  at: number;
  views?: number;
}

export interface AnalyticsDay {
  day: string;
  views: number;
  viewers: number;
}

export interface PageAnalytics {
  days: number;
  series: AnalyticsDay[];
  views: number;
  viewers: number;
  anonymous: number;
  created_at: number;
  created_by: AnalyticsPerson | null;
  updated_at: number;
  updated_by: AnalyticsPerson | null;
  editors: AnalyticsPerson[];
  people: AnalyticsPerson[];
  history: boolean;
}

export interface TopPage {
  id: string;
  title: string;
  icon: string;
  kind: PageKind;
  views: number;
  viewers: number;
}

export interface WorkspaceAnalytics {
  days: number;
  series: AnalyticsDay[];
  views: number;
  viewers: number;
  pages: number;
  top: TopPage[];
  history: boolean;
}

export interface SiteStatus {
  site: Site | null;
  can_publish: boolean;
  allowed: boolean;
  can_host: boolean;
  suggested_slug: string;
}

export interface SiteInput {
  slug?: string;
  title?: string;
  description?: string;
  image?: string;
  theme?: Site["theme"];
  font?: Site["font"];
  password?: string | null;
  indexing?: boolean;
  subpages?: boolean;
  hostname?: string;
}

export type NotificationKind ="mention" | "comment" | "reply" | "suggestion" | "decision" | "share" | "assign" | "reminder";

export interface InboxItem {
  id: string;
  workspace_id: string;
  kind: NotificationKind;
  page_id: string;
  page_icon: string;
  thread_id: string;
  actor_id: string;
  actor_name: string;
  actor_color: string;
  actor_avatar: string;
  title: string;
  body: string;
  created_at: number;
  read_at: number;
  archived_at: number;
}

export interface NotifyPrefs {
  email: "all" | "mentions" | "off";
  push_key: string;
  subscriptions: number;
  mail_enabled: boolean;
}

export interface Crumb {
  id: string;
  title: string;
  icon: string;
}

export interface PageDetail extends Page {
  path: Crumb[];
  can_write: boolean;
  level: AccessLevel;
  can_edit: boolean;
  can_comment: boolean;
  can_share: boolean;
  guest?: boolean;
}

export interface Favorite extends Page {
  favorite_key: string;
}

export interface RecentPage extends Page {
  visited_at: number;
}

export interface SearchQuery {
  q: string;
  workspace_id: string;
  in?: "title" | "all";
  creator?: string[];
  parent?: string;
  date?: "edited" | "created";
  from?: number;
  to?: number;
  sort?: "best" | "edited" | "created";
  limit?: number;
  offset?: number;
}

export interface SearchHit {
  id: string;
  title: string;
  title_hl: string;
  icon: string;
  kind: "doc" | "collection";
  row: boolean;
  path: { id: string; title: string; icon: string }[];
  match: "title" | "body" | "props";
  heading: string;
  snippet: string;
  created_at: number;
  created_by: string;
  updated_at: number;
}

export interface SearchResult {
  hits: SearchHit[];
  more: boolean;
  took_ms: number;
}

export interface TagInfo {
  tag: string;
  count: number;
  color: string;
}

export interface TrashItem extends Page {
  trashed_by_name: string;
  pages: number;
}

export type FilePurpose = "attachment" | "cover" | "icon" | "avatar" | "workspace";

export interface FileInfo {
  id: string;
  workspace_id: string;
  page_id: string;
  owner_id: string;
  purpose: FilePurpose;
  name: string;
  mime: string;
  size: number;
  created_at: number;
  url: string;
  ref: string;
  w?: number;
  h?: number;
}

export interface LinkPreview {
  url: string;
  kind: "page" | "image" | "video" | "audio" | "file";
  mime?: string;
  title: string;
  description: string;
  site: string;
  image: string;
  icon: string;
  w?: number;
  h?: number;
  frameable: boolean;
}

export type LiveEvent =
  | { type: "hello"; version: string; seq: number; reset: boolean }
  | { type: "ping"; seq: number }
  | { type: "page"; seq: number; workspace_id: string; page: Page }
  | { type: "page_removed"; seq: number; workspace_id: string; ids: string[] }
  | { type: "workspace"; seq: number; workspace_id: string; what: string }
  | { type: "user"; seq: number; what: string }
  | { type: "comments"; seq: number; workspace_id: string; page_id: string }
  | { type: "collection"; seq: number; workspace_id: string; collection_id: string; what: "rows" | "schema" | "views"; ids?: string[]; epoch: number };

export interface Member {
  user_id: string;
  name: string;
  email: string;
  color: string;
  avatar: string;
  role: WorkspaceRole;
  joined_at: number;
  disabled: boolean;
}

export interface Invite {
  id: string;
  email: string;
  role: WorkspaceRole;
  workspace_id: string;
  created_by: string;
  created_at: number;
  expires_at: number;
}

export interface InviteInfo {
  workspace: { name: string; icon: string };
  inviter: string;
  role: WorkspaceRole;
  email: string;
  account_exists: boolean;
  signed_in_as?: string;
  page?: { id: string; title: string; level: string };
}

export interface Token {
  id: string;
  name: string;
  prefix: string;
  scope: "read" | "write";
  all_workspaces: boolean;
  workspaces: string[];
  created_at: number;
  last_used_at: number;
  last_used_ip: string;
}

export interface SessionInfo {
  id: string;
  current: boolean;
  created_at: number;
  last_seen_at: number;
  user_agent: string;
  ip: string;
}

export interface Rules {
  rules: string;
  updated_at: number;
  proposal?: string;
  proposal_by?: string;
  proposal_by_name?: string;
  proposal_at?: number;
  can_edit: boolean;
}

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  color: string;
  avatar: string;
  role: InstanceRole;
  disabled: boolean;
  totp_enabled: boolean;
  email_verified: boolean;
  created_at: number;
  last_seen_at: number;
  workspaces: number;
}

export interface DeletionPlan {
  user_id: string;
  name: string;
  email: string;
  purge: { id: string; name: string }[];
  promote: { id: string; name: string; user_id: string; user_name: string }[];
  hand_to_owner: { id: string; name: string }[];
}

export interface BreakGlassGrant {
  id: string;
  workspace_id: string;
  workspace_name: string;
  reason: string;
  created_at: number;
  expires_at: number;
}

export interface EmergencyGrant {
  id: string;
  user_name: string;
  reason: string;
  created_at: number;
  expires_at: number;
  revoked_at: number;
}

export interface AccessWorkspace {
  id: string;
  name: string;
  icon: string;
  auto_join: boolean;
  agent_access: AgentAccess;
  members: { user_id: string; role: WorkspaceRole }[];
}

export interface StrandedWorkspace {
  id: string;
  name: string;
  created_at: number;
  members: number;
}

export type SettingsMap = Record<string, string>;
