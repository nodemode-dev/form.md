package agentlookup

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/nowindow"
)

const Version = "1.0.1"

type Program struct {
	Title          string
	Names          []string
	Setting        string
	Install        string
	InstallWindows string
	Home           []string
	Windows        []string
	MacFiles       []string
	WindowsSearch  []string
	Real           map[string]string
}

var Order = []string{"claude", "codex", "cursor", "antigravity"}

var Programs = map[string]Program{
	"claude": {Title: "Claude Code", Names: []string{"claude"}, Setting: "claudeBin",
		Install: "curl -fsSL https://claude.ai/install.sh | bash", InstallWindows: "irm https://claude.ai/install.ps1 | iex",
		Home: []string{".claude/local", ".claude/bin"}, Windows: []string{"home:.claude/local", "home:.claude/bin", "local:Programs/claude", "roaming:Programs/claude"}},
	"codex": {Title: "Codex", Names: []string{"codex"}, Setting: "codexBin", Install: "npm install -g @openai/codex",
		Windows: []string{"local:Programs/OpenAI/Codex/bin"}, MacFiles: []string{"Codex.app/Contents/Resources/codex", "Codex.app/Contents/Resources/bin/codex"}},
	"cursor": {Title: "Cursor", Names: []string{"cursor-agent", "agent"}, Setting: "cursorBin", Install: "curl https://cursor.com/install -fsS | bash",
		Home: []string{".local/share/cursor-agent/versions/*"}, Windows: []string{"local:cursor-agent"}, Real: map[string]string{"agent": "cursor"}},
	"antigravity": {Title: "Antigravity", Names: []string{"agy"}, Setting: "antigravityBin",
		Install: "curl -fsSL https://antigravity.google/cli/install.sh | bash", InstallWindows: "irm https://antigravity.google/cli/install.ps1 | iex",
		WindowsSearch: []string{"antigravity", "agy"}},
}

var KeepEnv = map[string]bool{"CLAUDE_CODE_OAUTH_TOKEN": true, "CLAUDE_CODE_USE_BEDROCK": true, "CLAUDE_CODE_USE_VERTEX": true, "CLAUDE_CONFIG_DIR": true}

var libraryVars = []string{"LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH"}

var windowsEndings = []string{".exe", ".cmd", ".bat"}

const mark = "__FORM_AGENT_LOOKUP__"

const envHint = "use -[v]S to pass options"

var how = map[string]string{"setting": "your setting", "override": "the development override", "path": "the engine's PATH", "folder": "a known install folder", "shell": "your shell's PATH"}

const retry = 60 * time.Second

type kept struct {
	state   string
	version string
	err     string
	at      time.Time
}

var (
	lock  sync.Mutex
	shell struct {
		set  bool
		path []string
		note string
		at   time.Time
	}
	checked = map[string]kept{}
)

type Found struct {
	Program string
	Title   string
	Path    string
	How     string
	State   string
	Version string
	Error   string
	Looked  []string
}

type Query struct {
	Setting      *string
	OverrideVars []string
	Env          []string
	Home         string
	Platform     string
	ShellTimeout time.Duration
	Fresh        bool
}

type Child struct {
	Path         string
	Base         []string
	Extra        []string
	Front        []string
	Home         string
	Platform     string
	ShellTimeout time.Duration
	Keep         map[string]bool
}

type Status struct {
	Installed bool     `json:"installed"`
	Path      *string  `json:"path"`
	Version   *string  `json:"version"`
	Finding   string   `json:"finding"`
	How       string   `json:"how"`
	Looked    []string `json:"looked"`
	Error     string   `json:"error"`
	Said      string   `json:"said"`
}

func Register(key string, p Program) {
	if _, ok := Programs[key]; !ok {
		Order = append(Order, key)
	}
	Programs[key] = p
}

func Install(program, platform string) string {
	spec := Programs[program]
	if Windows(platform) && spec.InstallWindows != "" {
		return spec.InstallWindows
	}
	return spec.Install
}

func Forget() {
	lock.Lock()
	defer lock.Unlock()
	shell.set, shell.path, shell.note = false, nil, ""
	checked = map[string]kept{}
}

func system(platform string) string {
	if platform == "" {
		return runtime.GOOS
	}
	return platform
}

func Windows(platform string) bool { return strings.HasPrefix(system(platform), "win") }

func environ(env []string) []string {
	if env == nil {
		return os.Environ()
	}
	return env
}

func get(env []string, key string) string {
	value := ""
	for _, kv := range env {
		if k, v, ok := strings.Cut(kv, "="); ok && k == key {
			value = v
		}
	}
	return value
}

func has(env []string, key string) bool {
	for _, kv := range env {
		if k, _, ok := strings.Cut(kv, "="); ok && k == key {
			return true
		}
	}
	return false
}

func homeOf(home string) string {
	if home != "" {
		return home
	}
	h, _ := os.UserHomeDir()
	return h
}

func expand(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		return filepath.Join(homeOf(""), path[1:])
	}
	return path
}

func Runnable(path, platform string) bool {
	st, err := os.Stat(expand(path))
	return err == nil && st.Mode().IsRegular() && (Windows(platform) || st.Mode()&0o111 != 0)
}

func Short(path, home string) string {
	base := homeOf(home)
	sep := string(os.PathSeparator)
	if base != "" && base != sep && (path == base || strings.HasPrefix(path, base+sep)) {
		return "~" + path[len(base):]
	}
	return path
}

var digits = regexp.MustCompile(`\d+`)

func versionKey(name string) []int {
	out := []int{}
	for _, d := range digits.FindAllString(name, -1) {
		n, _ := strconv.Atoi(d)
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int{0}
	}
	return out
}

func newer(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return len(a) > len(b)
}

func versions(base, tail string) []string {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	names := []string{}
	for _, e := range entries {
		if st, err := os.Stat(filepath.Join(base, e.Name())); err == nil && st.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.SliceStable(names, func(i, j int) bool { return newer(versionKey(names[i]), versionKey(names[j])) })
	out := []string{}
	for _, n := range names {
		out = append(out, filepath.Join(base, n, filepath.FromSlash(tail)))
	}
	return out
}

var SystemFolders = func(platform string) []string {
	platform = system(platform)
	if Windows(platform) {
		return nil
	}
	if platform == "darwin" {
		return []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin"}
	}
	return []string{"/usr/local/bin", "/usr/bin", "/snap/bin", "/home/linuxbrew/.linuxbrew/bin"}
}

func unique(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func Folders(program, home string, env []string, platform string) []string {
	platform, home, env = system(platform), homeOf(home), environ(env)
	win := Windows(platform)
	spec := Programs[program]
	out := []string{}
	for _, key := range []string{"NPM_CONFIG_PREFIX", "npm_config_prefix"} {
		if v := get(env, key); v != "" {
			if win {
				out = append(out, v)
			} else {
				out = append(out, filepath.Join(v, "bin"))
			}
		}
	}
	for _, pair := range [][2]string{{"PNPM_HOME", ""}, {"VOLTA_HOME", "bin"}, {"BUN_INSTALL", "bin"}, {"NVM_SYMLINK", ""}} {
		if v := get(env, pair[0]); v != "" {
			out = append(out, filepath.Join(v, pair[1]))
		}
	}
	if win {
		roots := map[string]string{"home": home, "roaming": or(get(env, "APPDATA"), filepath.Join(home, "AppData", "Roaming")), "local": or(get(env, "LOCALAPPDATA"), filepath.Join(home, "AppData", "Local"))}
		out = append(out, filepath.Join(home, ".local", "bin"), filepath.Join(roots["roaming"], "npm"), filepath.Join(roots["local"], "npm"), filepath.Join(roots["local"], "pnpm"),
			filepath.Join(roots["local"], "Volta", "bin"), filepath.Join(home, ".bun", "bin"), filepath.Join(home, ".volta", "bin"))
		for _, place := range spec.Windows {
			root, rest, _ := strings.Cut(place, ":")
			out = append(out, filepath.Join(roots[root], filepath.FromSlash(rest)))
		}
		return unique(out)
	}
	out = append(out, filepath.Join(home, ".local", "bin"), filepath.Join(home, ".npm-global", "bin"), filepath.Join(home, ".bun", "bin"), filepath.Join(home, ".volta", "bin"),
		filepath.Join(home, ".local", "share", "pnpm"))
	if platform == "darwin" {
		out = append(out, filepath.Join(home, "Library", "pnpm"))
	}
	out = append(out, versions(filepath.Join(home, ".nvm", "versions", "node"), "bin")...)
	out = append(out, versions(filepath.Join(home, ".local", "share", "fnm", "node-versions"), "installation/bin")...)
	out = append(out, versions(filepath.Join(home, "Library", "Application Support", "fnm", "node-versions"), "installation/bin")...)
	out = append(out, versions(filepath.Join(home, ".asdf", "installs", "nodejs"), "bin")...)
	out = append(out, versions(filepath.Join(home, ".local", "share", "mise", "installs", "node"), "bin")...)
	out = append(out, filepath.Join(home, ".asdf", "shims"), filepath.Join(home, ".local", "share", "mise", "shims"))
	for _, place := range spec.Home {
		base, tail, star := strings.Cut(place, "*")
		if star {
			out = append(out, versions(filepath.Join(home, filepath.FromSlash(strings.Trim(base, "/"))), strings.Trim(tail, "/"))...)
		} else {
			out = append(out, filepath.Join(home, filepath.FromSlash(place)))
		}
	}
	out = append(out, SystemFolders(platform)...)
	return unique(out)
}

func subfolders(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, e := range entries {
		if st, err := os.Stat(filepath.Join(root, e.Name())); err == nil && st.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func searched(root string, fragments []string) []string {
	named := func(name string) bool {
		for _, f := range fragments {
			if strings.Contains(strings.ToLower(name), f) {
				return true
			}
		}
		return false
	}
	out := []string{}
	for _, one := range subfolders(root) {
		if named(one) {
			out = append(out, filepath.Join(root, one))
		}
		for _, two := range subfolders(filepath.Join(root, one)) {
			if named(one) || named(two) {
				out = append(out, filepath.Join(root, one, two))
			}
		}
	}
	return out
}

func Candidates(program, home string, env []string, platform string) []string {
	platform, home, env = system(platform), homeOf(home), environ(env)
	spec := Programs[program]
	endings := []string{""}
	if Windows(platform) {
		endings = windowsEndings
	}
	places := Folders(program, home, env, platform)
	if Windows(platform) && len(spec.WindowsSearch) > 0 {
		places = append(places, searched(or(get(env, "LOCALAPPDATA"), filepath.Join(home, "AppData", "Local")), spec.WindowsSearch)...)
	}
	out := []string{}
	for _, folder := range unique(places) {
		for _, name := range spec.Names {
			for _, ending := range endings {
				out = append(out, filepath.Join(folder, name+ending))
			}
		}
	}
	if platform == "darwin" {
		for _, apps := range []string{"/Applications", filepath.Join(home, "Applications")} {
			for _, inside := range spec.MacFiles {
				out = append(out, filepath.Join(apps, filepath.FromSlash(inside)))
			}
		}
	}
	return out
}

func trusted(program, path string) bool {
	name := strings.ToLower(filepath.Base(path))
	stem := name
	for _, ending := range windowsEndings {
		if strings.HasSuffix(name, ending) {
			stem = name[:strings.LastIndex(name, ".")]
		}
	}
	word := Programs[program].Real[stem]
	if word == "" {
		return true
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	return strings.Contains(strings.ToLower(abs), word)
}

func which(name, path, platform string) string {
	endings := []string{""}
	if Windows(platform) {
		endings = append([]string{""}, windowsEndings...)
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}
		for _, ending := range endings {
			if p := filepath.Join(dir, name+ending); Runnable(p, platform) {
				return p
			}
		}
	}
	return ""
}

func onPath(program, path, platform string) string {
	if path == "" {
		return ""
	}
	for _, name := range Programs[program].Names {
		if hit := which(name, path, platform); hit != "" && trusted(program, hit) {
			if abs, err := filepath.Abs(hit); err == nil {
				return abs
			}
			return hit
		}
	}
	return ""
}

func LoginShell(env []string, platform string) string {
	env = environ(env)
	if chosen := get(env, "SHELL"); chosen != "" {
		return chosen
	}
	if system(platform) == "darwin" {
		if _, err := os.Stat("/bin/zsh"); err == nil {
			return "/bin/zsh"
		}
	}
	if p, err := exec.LookPath("bash"); err == nil {
		return p
	}
	return "/bin/sh"
}

func seconds(d time.Duration) string { return strconv.FormatFloat(d.Seconds(), 'g', -1, 64) }

func askShell(sh string, flags []string, env []string, timeout time.Duration) []string {
	script := `printf "%s%s%s" ` + mark + ` "$PATH" ` + mark
	if filepath.Base(sh) == "fish" {
		script = "printf '%s%s%s' " + mark + " (string join : $PATH) " + mark
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := nowindow.CommandContext(ctx, sh, append(append([]string{}, flags...), "-c", script)...)
	cmd.Env = env
	ownSession(cmd)
	cmd.WaitDelay = 2 * time.Second
	out, _ := cmd.Output()
	if ctx.Err() != nil {
		return nil
	}
	text := string(out)
	start := strings.Index(text, mark)
	if start < 0 {
		return nil
	}
	rest := text[start+len(mark):]
	end := strings.Index(rest, mark)
	if end < 0 {
		return nil
	}
	parts := []string{}
	for _, p := range filepath.SplitList(rest[:end]) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return parts
}

func ShellPath(env []string, timeout time.Duration, fresh bool, platform string) ([]string, string) {
	env = environ(env)
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	lock.Lock()
	defer lock.Unlock()
	if shell.set && !fresh && (len(shell.path) > 0 || time.Since(shell.at) < retry) {
		return append([]string{}, shell.path...), shell.note
	}
	var path []string
	note := ""
	if Windows(platform) {
		path = registryPath()
		note = "the Path values in the registry"
		if len(path) == 0 {
			note = "the Path values in the registry (none could be read)"
		}
	} else {
		sh := LoginShell(env, platform)
		name := filepath.Base(sh)
		path = askShell(sh, []string{"-l", "-i"}, env, timeout)
		note = "your shell (" + name + ", login and interactive)"
		if path == nil {
			path = askShell(sh, []string{"-l"}, env, timeout)
			note = "your shell (" + name + ", login only: the interactive start gave no PATH within " + seconds(timeout) + " s)"
		}
		if path == nil {
			note = "your shell (" + name + " gave no PATH within " + seconds(timeout) + " s)"
		}
	}
	shell.set, shell.path, shell.note, shell.at = true, append([]string{}, path...), note, time.Now()
	return append([]string{}, path...), note
}

func Find(program string, q Query) Found {
	env := environ(q.Env)
	platform := system(q.Platform)
	spec := Programs[program]
	found := Found{Program: program, Title: spec.Title, State: "missing", Looked: []string{}}
	at := func(path, by string) Found {
		found.Path, found.How, found.State = path, by, "found"
		return found
	}
	if q.Setting != nil {
		chosen := strings.TrimSpace(*q.Setting)
		if chosen != "" && Runnable(chosen, platform) {
			return at(expand(chosen), "setting")
		}
		if chosen != "" {
			found.Looked = append(found.Looked, "your setting ("+chosen+", which is not a program that can run)")
		} else {
			found.Looked = append(found.Looked, "your setting (empty)")
		}
	}
	for _, name := range q.OverrideVars {
		value := strings.TrimSpace(get(env, name))
		if value != "" && Runnable(value, platform) {
			return at(expand(value), "override")
		}
		if value != "" {
			found.Looked = append(found.Looked, name+" ("+value+", which is not a program that can run)")
		}
	}
	enginePath := get(env, "PATH")
	if hit := onPath(program, enginePath, platform); hit != "" {
		return at(hit, "path")
	}
	found.Looked = append(found.Looked, "PATH ("+or(enginePath, "empty")+")")
	places := Candidates(program, q.Home, env, platform)
	for _, place := range places {
		if Runnable(place, platform) && trusted(program, place) {
			return at(place, "folder")
		}
	}
	shown := []string{}
	for _, place := range places {
		shown = append(shown, Short(filepath.Dir(place), q.Home))
	}
	found.Looked = append(found.Looked, strings.Join(unique(shown), ", "))
	dirs, note := ShellPath(env, q.ShellTimeout, q.Fresh, platform)
	if hit := onPath(program, strings.Join(dirs, string(os.PathListSeparator)), platform); hit != "" {
		return at(hit, "shell")
	}
	found.Looked = append(found.Looked, note)
	return found
}

func ChildEnv(c Child) []string {
	base := environ(c.Base)
	keep := c.Keep
	if keep == nil {
		keep = KeepEnv
	}
	drop := map[string]bool{"PATH": true}
	put := []string{}
	for _, name := range libraryVars {
		if has(base, name+"_ORIG") {
			drop[name] = true
			put = append(put, name+"="+get(base, name+"_ORIG"))
		} else if has(base, "_MEIPASS2") {
			drop[name] = true
		}
	}
	out := []string{}
	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if drop[k] || (strings.HasPrefix(k, "CLAUDE") && !keep[k]) {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, put...)
	dirs, _ := ShellPath(base, c.ShellTimeout, false, c.Platform)
	parts := []string{}
	if c.Path != "" {
		parts = append(parts, filepath.Dir(c.Path))
	}
	parts = append(parts, c.Front...)
	parts = append(parts, dirs...)
	for _, program := range Order {
		for _, f := range Folders(program, c.Home, base, c.Platform) {
			if st, err := os.Stat(f); err == nil && st.IsDir() {
				parts = append(parts, f)
			}
		}
	}
	parts = append(parts, filepath.SplitList(get(base, "PATH"))...)
	clean := []string{}
	for _, p := range unique(parts) {
		if p != "" {
			clean = append(clean, p)
		}
	}
	out = append(out, "PATH="+strings.Join(clean, string(os.PathListSeparator)))
	return append(out, c.Extra...)
}

var plain = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

func quote(word string, fish bool) string {
	if plain.MatchString(word) {
		return word
	}
	if fish {
		return "'" + strings.ReplaceAll(strings.ReplaceAll(word, `\`, `\\`), "'", `\'`) + "'"
	}
	return "'" + strings.ReplaceAll(word, "'", `'"'"'`) + "'"
}

func StartArgv(cmd []string, env []string, platform string) []string {
	env = environ(env)
	if Windows(platform) {
		first := strings.ToLower(cmd[0])
		if strings.HasSuffix(first, ".cmd") || strings.HasSuffix(first, ".bat") {
			return append([]string{or(get(env, "COMSPEC"), "cmd.exe"), "/c"}, cmd...)
		}
		return append([]string{}, cmd...)
	}
	sh := LoginShell(env, platform)
	fish := filepath.Base(sh) == "fish"
	words := make([]string, len(cmd))
	for i, w := range cmd {
		words[i] = quote(w, fish)
	}
	return []string{sh, "-l", "-c", "exec " + strings.Join(words, " ")}
}

type Result struct {
	Code   int
	Stdout string
	Stderr string
}

var ErrTimeout = errors.New("no answer in time")

func Run(cmd []string, env []string, timeout time.Duration, platform string) (Result, error) {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	argv := StartArgv(cmd, env, platform)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := nowindow.CommandContext(ctx, argv[0], argv[1:]...)
	c.Env = env
	var stdout, stderr strings.Builder
	c.Stdout, c.Stderr = &stdout, &stderr
	c.WaitDelay = 2 * time.Second
	err := c.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() != nil {
		return res, ErrTimeout
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		res.Code = exit.ExitCode()
		return res, nil
	}
	return res, err
}

func lastLine(text string) string {
	last := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, envHint) {
			last = line
		}
	}
	if r := []rune(last); len(r) > 300 {
		last = string(r[:300])
	}
	return last
}

func version(text string) string {
	words := strings.Fields(text)
	for _, w := range words {
		if strings.ContainsAny(w, "0123456789") {
			return w
		}
	}
	if len(words) > 0 {
		return words[0]
	}
	return ""
}

func stamp(path string) string {
	st, err := os.Stat(path)
	if err != nil {
		return path + "|0|0"
	}
	return path + "|" + strconv.FormatInt(st.ModTime().UnixNano(), 10) + "|" + strconv.FormatInt(st.Size(), 10)
}

func Check(found Found, env []string, timeout time.Duration, fresh bool, platform string) Found {
	if found.Path == "" {
		return found
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	key := stamp(found.Path)
	lock.Lock()
	had, ok := checked[key]
	lock.Unlock()
	if fresh || (ok && had.state != "runs" && time.Since(had.at) >= retry) {
		ok = false
	}
	if !ok {
		if env == nil {
			env = ChildEnv(Child{Path: found.Path, Platform: platform})
		}
		res, err := Run([]string{found.Path, "--version"}, env, timeout, platform)
		had = kept{state: "broken", at: time.Now()}
		switch {
		case errors.Is(err, ErrTimeout):
			had.err = "it gave no answer to --version in " + seconds(timeout) + " seconds"
		case err != nil:
			had.err = err.Error()
			var pathErr *os.PathError
			if errors.As(err, &pathErr) {
				had.err = pathErr.Err.Error()
			}
		case res.Code == 0:
			had.state, had.version = "runs", version(res.Stdout)
		default:
			had.err = or(or(lastLine(res.Stderr), lastLine(res.Stdout)), "it ended with code "+strconv.Itoa(res.Code))
		}
		lock.Lock()
		checked[key] = had
		lock.Unlock()
	}
	found.State, found.Version, found.Error = had.state, had.version, had.err
	return found
}

func Sentence(found Found, install, otherwise string) string {
	if found.State == "missing" {
		ways := []string{}
		if install != "" {
			ways = append(ways, "install it with: "+install)
		}
		if otherwise != "" {
			ways = append(ways, otherwise)
		}
		tail := ""
		if len(ways) > 0 {
			joined := strings.Join(ways, ", or ") + "."
			tail = " " + strings.ToUpper(joined[:1]) + joined[1:]
		}
		return found.Title + " was not found. Looked in: " + strings.Join(found.Looked, "; ") + "." + tail
	}
	where := found.Path + " (" + or(how[found.How], found.How) + ")"
	if found.State == "broken" {
		tail := ""
		if otherwise != "" {
			tail = " To use another copy, " + otherwise + "."
		}
		return found.Title + " was found at " + where + " but does not start: " + found.Error + "." + tail
	}
	return found.Title + " is at " + where + "."
}

func Answer(found Found, install, otherwise string) Status {
	out := Status{Installed: found.State == "runs", Finding: found.State, How: found.How, Looked: append([]string{}, found.Looked...), Error: found.Error,
		Said: Sentence(found, install, otherwise)}
	if found.Path != "" {
		p := found.Path
		out.Path = &p
	}
	if found.Version != "" {
		v := found.Version
		out.Version = &v
	}
	return out
}
