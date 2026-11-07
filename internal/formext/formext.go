package formext

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const API = "1"

var (
	idRE      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,63}$`)
	localRE   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	versionRE = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

var Types = map[string]string{
	".html":  "text/html; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".webp":  "image/webp",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".wasm":  "application/wasm",
	".txt":   "text/plain; charset=utf-8",
	".md":    "text/markdown; charset=utf-8",
	".wav":   "audio/wav",
	".ogg":   "audio/ogg",
	".mp3":   "audio/mpeg",
	".webm":  "video/webm",
	".mp4":   "video/mp4",
	".cube":  "text/plain; charset=utf-8",
	".woff2": "font/woff2",
}

var Permissions = map[string]bool{"document": true, "files-read": true, "files-write": true, "network": true, "models": true, "native-code": true}

func PortalRoot() string {
	if env := strings.TrimSpace(os.Getenv("FORMPORTAL_ROOT")); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		support := filepath.Join(home, "Library", "Application Support")
		next, old := filepath.Join(support, "studio.formai.portal"), filepath.Join(support, "dev.formai.portal")
		if isDir(old) && !exists(next) {
			return old
		}
		return next
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "FormPortal")
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "formportal")
}

func VersionTuple(v string) [3]int {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		parts = parts[:3]
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}
		}
		out[i] = n
	}
	return out
}

func newer(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

type Panel struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Location string `json:"location"`
}

type Command struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Panel string `json:"panel,omitempty"`
}

type Extension struct {
	ID          string
	Version     string
	Source      string
	Path        string
	Name        string
	Kind        string
	Description string
	Permissions []string
	EntryUI     string
	Panels      []Panel
	Commands    []Command
	State       string
	Reason      string
}

type Public struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Kind        string    `json:"kind"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	Base        string    `json:"base"`
	EntryUI     string    `json:"entryUi"`
	Panels      []Panel   `json:"panels"`
	Commands    []Command `json:"commands"`
	Path        *string   `json:"path"`
	State       string    `json:"state"`
	Reason      string    `json:"reason"`
}

func (e *Extension) Public(base string) Public {
	name := e.Name
	if name == "" {
		name = e.ID
	}
	var path *string
	if e.Source == "development" {
		p := e.Path
		path = &p
	}
	return Public{
		ID: e.ID, Name: name, Version: e.Version, Kind: e.Kind, Source: e.Source, Description: e.Description,
		Permissions: orEmpty(e.Permissions), Base: fmt.Sprintf("%s/%s/%s/", base, e.ID, e.Version), EntryUI: e.EntryUI,
		Panels: orEmptyPanels(e.Panels), Commands: orEmptyCommands(e.Commands), Path: path, State: e.State, Reason: e.Reason,
	}
}

func readJSON(path string) any {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return v
}

func safeRel(rel string) bool {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, "\\") || len(rel) > 200 {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func Check(ext *Extension, product, appVersion, appName string) *Extension {
	refuse := func(reason string) *Extension {
		ext.State, ext.Reason = "refused", reason
		return ext
	}

	manifest, ok := readJSON(filepath.Join(ext.Path, "manifest.json")).(map[string]any)
	if !ok {
		return refuse("its manifest.json is missing or not valid JSON")
	}
	if manifest["format"] != "formai-package" || manifest["format_version"] != float64(1) {
		return refuse("it is not a formai package")
	}
	ext.Name = cut(orText(manifest["name"], ext.ID), 80)
	ext.Kind = orText(manifest["kind"], "")
	ext.Description = cut(orText(manifest["description"], ""), 400)
	if ext.Source == "development" && versionRE.MatchString(pyStrDefault(manifest, "version")) {
		ext.Version = pyStr(manifest["version"])
	}
	if manifest["id"] != ext.ID {
		return refuse(fmt.Sprintf("its manifest says %s, not %s", repr(manifest["id"]), repr(ext.ID)))
	}
	if manifest["product"] != product {
		return refuse(fmt.Sprintf("it is made for %s, not %s", pyStr(manifest["product"]), appName))
	}
	if pyStrDefault(manifest, "version") != ext.Version && ext.Source != "development" {
		return refuse(fmt.Sprintf("its manifest says version %s, FormPortal says %s", pyStr(manifest["version"]), ext.Version))
	}
	ext.Version = orText(manifest["version"], ext.Version)
	if !versionRE.MatchString(ext.Version) {
		return refuse("its version is not x.y.z")
	}
	if manifest["api"] == nil {
		return refuse(fmt.Sprintf("it does not say which extension API it needs; this %s offers API %s", appName, API))
	}
	if manifest["api"] != API {
		return refuse(fmt.Sprintf("this extension needs a newer %s (it asks for extension API %s; this %s offers API %s)", appName, pyStr(manifest["api"]), appName, API))
	}
	if truthy(manifest["native"]) {
		return refuse(fmt.Sprintf("native extensions are not supported by %s", appName))
	}
	perms := []any{}
	if truthy(manifest["permissions"]) {
		list, ok := manifest["permissions"].([]any)
		if !ok {
			return refuse("it asks for a permission that does not exist")
		}
		perms = list
	}
	ext.Permissions = []string{}
	seen := map[string]bool{}
	for _, p := range perms {
		s, ok := p.(string)
		if !ok || !Permissions[s] {
			return refuse("it asks for a permission that does not exist")
		}
		if !seen[s] {
			seen[s] = true
			ext.Permissions = append(ext.Permissions, s)
		}
	}
	if minimum := manifest["minimum_app"]; truthy(minimum) && newer(VersionTuple(pyStr(minimum)), VersionTuple(appVersion)) {
		return refuse(fmt.Sprintf("it needs %s %s or newer", appName, pyStr(minimum)))
	}
	entryObj, _ := manifest["entry"].(map[string]any)
	entry := ""
	if raw, ok := entryObj["ui"]; ok {
		text, isText := raw.(string)
		if !isText {
			return refuse("its entry.ui is not a path inside the package")
		}
		entry = text
	}
	contributes, _ := manifest["contributes"].(map[string]any)
	var panels []map[string]any
	var commands []map[string]any
	for _, raw := range asList(contributes["panels"]) {
		if p, ok := raw.(map[string]any); ok && localRE.MatchString(pyStrDefault(p, "id")) && truthy(p["title"]) {
			panels = append(panels, p)
		}
	}
	for _, raw := range asList(contributes["commands"]) {
		if c, ok := raw.(map[string]any); ok && localRE.MatchString(pyStrDefault(c, "id")) && truthy(c["title"]) {
			commands = append(commands, c)
		}
	}
	if (len(panels) > 0 || len(commands) > 0) && entry == "" {
		return refuse("it contributes panels or commands but has no entry.ui")
	}
	if entry != "" {
		if !safeRel(entry) {
			return refuse("its entry.ui is not a path inside the package")
		}
		target := resolve(filepath.Join(ext.Path, entry))
		st, err := os.Stat(target)
		if err != nil || !st.Mode().IsRegular() || !within(resolve(ext.Path), target) {
			return refuse(fmt.Sprintf("its entry.ui %s is not in the package", entry))
		}
	}
	ext.EntryUI = entry
	ext.Panels = []Panel{}
	for _, p := range panels {
		loc, _ := p["location"].(string)
		if loc != "dock" && loc != "dialog" {
			loc = "dock"
		}
		ext.Panels = append(ext.Panels, Panel{ID: p["id"].(string), Title: cut(pyStr(p["title"]), 60), Location: loc})
	}
	ext.Commands = []Command{}
	for _, c := range commands {
		cmd := Command{ID: c["id"].(string), Title: cut(pyStr(c["title"]), 80)}
		if truthy(c["panel"]) {
			cmd.Panel = pyStr(c["panel"])
		}
		ext.Commands = append(ext.Commands, cmd)
	}
	return ext
}

func Resolve(product, appVersion, appName, root string) []*Extension {
	if root == "" {
		root = PortalRoot()
	}
	folder := filepath.Join(root, "extensions", product)
	found := map[string]*Extension{}
	if enabled, ok := readJSON(filepath.Join(folder, "enabled.json")).(map[string]any); ok && enabled["format"] == "formportal-enabled-extensions" {
		for _, raw := range asList(enabled["extensions"]) {
			row, ok := raw.(map[string]any)
			if !ok || !idRE.MatchString(pyStrDefault(row, "id")) {
				continue
			}
			source := "catalogue"
			if row["source"] == "sideload" {
				source = "sideload"
			}
			ext := &Extension{ID: row["id"].(string), Version: pyStrDefault(row, "version"), Source: source, Path: pyStrDefault(row, "path"), State: "ready"}
			found[ext.ID] = Check(ext, product, appVersion, appName)
		}
	}
	if dev, ok := readJSON(filepath.Join(folder, "dev.json")).(map[string]any); ok && dev["format"] == "formext-dev-extensions" {
		for _, raw := range asList(dev["extensions"]) {
			row, ok := raw.(map[string]any)
			if !ok || !idRE.MatchString(pyStrDefault(row, "id")) {
				continue
			}
			ext := &Extension{ID: row["id"].(string), Version: "0.0.0", Source: "development", Path: pyStrDefault(row, "path"), State: "ready"}
			found[ext.ID] = Check(ext, product, appVersion, appName)
		}
	}
	out := make([]*Extension, 0, len(found))
	for _, e := range found {
		out = append(out, e)
	}
	key := func(e *Extension) string {
		if e.Name != "" {
			return strings.ToLower(e.Name)
		}
		return strings.ToLower(e.ID)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if key(out[i]) != key(out[j]) {
			return key(out[i]) < key(out[j])
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func Stamp(product, root string) [2]float64 {
	if root == "" {
		root = PortalRoot()
	}
	folder := filepath.Join(root, "extensions", product)
	var out [2]float64
	for i, name := range []string{"enabled.json", "dev.json"} {
		if st, err := os.Stat(filepath.Join(folder, name)); err == nil {
			out[i] = float64(st.ModTime().UnixNano()) / 1e9
		}
	}
	return out
}

func CSPFor(base string) string {
	src := "'self'"
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") || strings.HasPrefix(base, "formext://") {
		src = base
	}
	return "default-src 'none'; script-src " + src + " 'wasm-unsafe-eval'; style-src " + src + " 'unsafe-inline'; " +
		"img-src " + src + " data: blob:; media-src " + src + " data: blob:; font-src " + src + "; connect-src 'none'; " +
		"frame-src 'none'; form-action 'none'; base-uri 'none'"
}

func FileFor(extensions []*Extension, id, version, rel string) (string, string, bool) {
	var ext *Extension
	for _, e := range extensions {
		if e.ID == id && e.Version == version && e.State == "ready" {
			ext = e
			break
		}
	}
	if ext == nil || !safeRel(rel) {
		return "", "", false
	}
	kind, ok := Types[suffix(rel)]
	if !ok {
		return "", "", false
	}
	root := resolve(ext.Path)
	candidate := filepath.Join(ext.Path, filepath.FromSlash(rel))
	if linked(ext.Path, rel) {
		return "", "", false
	}
	target, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", "", false
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", "", false
	}
	st, err := os.Stat(target)
	if err != nil || !within(root, target) || !st.Mode().IsRegular() {
		return "", "", false
	}
	return target, kind, true
}

func HeadersFor(contentType, base string) map[string]string {
	h := map[string]string{"X-Content-Type-Options": "nosniff", "Cache-Control": "no-store", "Cross-Origin-Resource-Policy": "cross-origin", "Access-Control-Allow-Origin": "*"}
	if strings.HasPrefix(contentType, "text/html") {
		h["Content-Security-Policy"] = CSPFor(base)
	}
	return h
}

func linked(base, rel string) bool {
	at := base
	for _, part := range strings.Split(rel, "/") {
		at = filepath.Join(at, part)
		if st, err := os.Lstat(at); err == nil && st.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func suffix(rel string) string {
	name := rel[strings.LastIndex(rel, "/")+1:]
	i := strings.LastIndex(name, ".")
	if i <= 0 || i == len(name)-1 {
		return ""
	}
	return strings.ToLower(name[i:])
}

func resolve(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

func pyStr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	raw, _ := json.Marshal(v)
	return string(raw)
}

func pyStrDefault(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	return pyStr(v)
}

func repr(v any) string {
	if s, ok := v.(string); ok {
		return "'" + s + "'"
	}
	return pyStr(v)
}

func orText(v any, fallback string) string {
	if truthy(v) {
		return pyStr(v)
	}
	return fallback
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func asList(v any) []any {
	list, _ := v.([]any)
	return list
}

func orEmpty(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func orEmptyPanels(v []Panel) []Panel {
	if v == nil {
		return []Panel{}
	}
	return v
}

func orEmptyCommands(v []Command) []Command {
	if v == nil {
		return []Command{}
	}
	return v
}
