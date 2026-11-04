package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/5uck1ess/devkit/lib"
	"github.com/spf13/cobra"
)

// guardDefaultStaleTTL mirrors sessionStaleTTL in src/mcp/tools.go so an
// idle-but-not-cleared session.json is never enforced against.
const guardDefaultStaleTTL = 30 * time.Minute

// guardExit is the exit-code contract the hook relies on:
//
//	0 — allow (PreToolUse), or emit approve JSON (Stop).
//	2 — hard block (PreToolUse). Diagnostic on stderr.
//
// Stop mode always exits 0; the verdict is carried in the JSON payload.
// We route every exit through this function so tests can stub it and
// cobra's own error-to-exit-1 mapping never leaks through.
var guardExit = os.Exit

// guardStdin / guardStdout / guardStderr are package-level IO hooks so
// tests can drive the command without touching os.* globals.
var (
	guardStdin  io.Reader = os.Stdin
	guardStdout io.Writer = os.Stdout
	guardStderr io.Writer = os.Stderr
)

var (
	guardToolName string
	guardStopMode bool
)

var guardCmd = &cobra.Command{
	Use:   "guard",
	Short: "PreToolUse/Stop hook enforcement for devkit workflows",
	Long: `Guard enforces workflow step ordering for the devkit engine.

In PreToolUse mode (default) it reads a tool name and exits 0 to allow
or 2 to block. In --stop mode it emits a Stop-hook JSON verdict on
stdout ({"decision":"approve"} or {"decision":"block","reason":...}).

The allowlist policy:
  command step + hard  → devkit MCP + TodoWrite + Skill
  prompt step  + hard  → read-only evidence tools + devkit MCP + Skill
                         + Agent/Task dispatch + companion-rescue Bash
  prompt step  + soft  → allow with a stderr nudge
  parallel / unknown   → allow (engine is dispatching)
  stale session (TTL)  → allow with a stderr warning (orphan recovery)
  cross-repo session   → allow silently (block remains in origin repo)

Skill is allowed under both step types so a workflow that's mid-run can
still load a nested skill (e.g. user asks for tri-review during a feature
workflow). The dispatched skill calls devkit_start, which the engine
either accepts (if the current session is reclaimable) or rejects with
a clear error — the guard does not need to second-guess that.`,
	// Override the root's PersistentPreRunE: the guard hook runs on
	// every PreToolUse call and must not require a git repo, must not
	// open the SQLite DB, and must not fail if the host project has no
	// .git at all. Same reason applies to the post-run hook.
	PersistentPreRunE:  func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error { return nil },
	Run:                runGuard,
	Args:               cobra.NoArgs,
	SilenceUsage:       true,
	SilenceErrors:      true,
}

func init() {
	guardCmd.Flags().StringVar(&guardToolName, "tool-name", "", "tool name (parsed from stdin JSON if empty)")
	guardCmd.Flags().BoolVar(&guardStopMode, "stop", false, "emit Stop-hook JSON verdict instead of PreToolUse exit code")
	rootCmd.AddCommand(guardCmd)
}

func runGuard(cmd *cobra.Command, args []string) {
	if guardStopMode {
		runStopGuard()
		return
	}
	runPreToolGuard()
}

// readToolInvocationFromStdin extracts tool_name and (for Bash)
// tool_input.command from a PreToolUse payload. Returns ("", "", nil)
// for genuinely empty stdin (the common case when --tool-name is used
// instead). Any real failure (read error, malformed JSON) returns a
// non-nil error so the caller can log it distinctly — silent "" on
// parse failure would mask Claude Code schema drift. Under hard
// enforcement the empty tool name still falls through to the default-
// block branch, so errors never cause a silent ALLOW.
//
// command is populated only for tools that ship a "command" field in
// tool_input (Bash). Other tools get "" here — callers that care about
// specific input shapes must parse stdin themselves.
func readToolInvocationFromStdin() (name, command string, err error) {
	data, rerr := io.ReadAll(guardStdin)
	if rerr != nil {
		return "", "", fmt.Errorf("read stdin: %w", rerr)
	}
	if len(data) == 0 {
		return "", "", nil
	}
	var payload struct {
		ToolName  string `json:"tool_name"`
		ToolInput struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}
	if uerr := json.Unmarshal(data, &payload); uerr != nil {
		return "", "", fmt.Errorf("parse PreToolUse JSON: %w", uerr)
	}
	return payload.ToolName, payload.ToolInput.Command, nil
}

// resolveDataDir returns CLAUDE_PLUGIN_DATA or "" if unset. The shell
// hook prints a one-line "enforcement disabled" diagnostic to stderr
// when this is empty; we mirror that so log output is unchanged.
func resolveDataDir(stopMode bool) string {
	dir := os.Getenv("CLAUDE_PLUGIN_DATA")
	if dir == "" {
		prefix := "devkit-guard"
		if stopMode {
			prefix = "devkit-stop-guard"
		}
		fmt.Fprintf(guardStderr, "%s: CLAUDE_PLUGIN_DATA unset — enforcement disabled\n", prefix)
	}
	return dir
}

// staleTTL honours the DEVKIT_SESSION_STALE_TTL_SECONDS override the
// old lib/read-session.sh supported, so operators who tuned the env
// var during the PR #64 rollout do not see behaviour drift. Garbage
// values (non-numeric, non-positive, whitespace) log a one-line warning
// to stderr and fall back to the default — silent downgrade to "never
// stale" would disable orphan recovery without any user-visible signal.
func staleTTL() time.Duration {
	raw := os.Getenv("DEVKIT_SESSION_STALE_TTL_SECONDS")
	if raw == "" {
		return guardDefaultStaleTTL
	}
	trimmed := strings.TrimSpace(raw)
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		fmt.Fprintf(guardStderr,
			"devkit-guard: DEVKIT_SESSION_STALE_TTL_SECONDS=%q is not an integer — using default %s\n",
			raw, guardDefaultStaleTTL)
		return guardDefaultStaleTTL
	}
	if n <= 0 {
		fmt.Fprintf(guardStderr,
			"devkit-guard: DEVKIT_SESSION_STALE_TTL_SECONDS=%d must be positive — using default %s\n",
			n, guardDefaultStaleTTL)
		return guardDefaultStaleTTL
	}
	return time.Duration(n) * time.Second
}

// currentRepoRoot resolves the absolute repo root the current Claude
// Code session is operating in, for the stop-guard scope check (issue
// #91). Prefer CLAUDE_PROJECT_DIR because Claude Code sets it per
// session for every hook invocation and it is not spoofable from
// within a prompt — Claude cannot edit its own session env. Fall back
// to walking up from pwd so a non-Claude-Code caller (direct stdio
// test) still resolves something sensible. Returns "" if no repo root
// can be determined; callers fall through to the legacy block in that
// case so a missing signal never silently approves.
func currentRepoRoot() string {
	if dir := strings.TrimSpace(os.Getenv("CLAUDE_PROJECT_DIR")); dir != "" {
		return dir
	}
	if dir, err := os.Getwd(); err == nil {
		for {
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return ""
			}
			dir = parent
		}
	}
	return ""
}

// samePath compares two filesystem paths by resolving symlinks and
// cleaning separators. Used by the stop-guard repo-match check so
// symlinked checkouts ("/Users/me/repos/x" via "/tmp/x") resolve
// equal. Falls back to cleaned-path comparison if EvalSymlinks fails
// (path does not exist yet, permission denied) so the check never
// panics on transient filesystem state.
func samePath(a, b string) bool {
	norm := func(p string) string {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Clean(resolved)
		}
		return filepath.Clean(p)
	}
	return norm(a) == norm(b)
}

// sessionIsStale mirrors the lib/read-session.sh TTL rule: prefer
// UpdatedAt, fall back to StartedAt for pre-UpdatedAt state files
// written by older engine binaries. An unparseable (zero) timestamp is
// treated as fresh so a schema gap can't silently disarm enforcement —
// but a zero-timestamp session is anomalous, so we surface it on stderr
// so a wedged session at least leaves a debuggable trail.
func sessionIsStale(s *lib.SessionState) bool {
	ref := s.UpdatedAt
	if ref.IsZero() {
		ref = s.StartedAt
	}
	if ref.IsZero() {
		fmt.Fprintf(guardStderr,
			"devkit-guard: WARNING session %s has no updated_at/started_at timestamps — stale-check disabled (engine version mismatch?)\n",
			s.Workflow)
		return false
	}
	return time.Since(ref) > staleTTL()
}

// stepLabel reproduces the shell step_label() helper so veto messages
// carry workflow + position + step id without another devkit_status
// round trip.
func stepLabel(s *lib.SessionState) string {
	if s.TotalSteps > 0 {
		return fmt.Sprintf("%s step %d/%d (%s)", s.Workflow, s.CurrentIndex+1, s.TotalSteps, s.CurrentStep)
	}
	return fmt.Sprintf("%s (%s)", s.Workflow, s.CurrentStep)
}

// sessionFileExists is the cheap read-only probe we use BEFORE touching
// the session lock. The no-workflow hot path (every PreToolUse call
// when the user has no active session) must not trigger the mkdir +
// lock-file create side effects of lib.ReadSessionJSON — on network
// mounts those writes can blow the 2s hook timeout and, more subtly,
// they materialise sentinel files in $CLAUDE_PLUGIN_DATA for users who
// never asked for a workflow. A single os.Stat is ~50µs locally and
// has zero filesystem side effects.
func sessionFileExists(dataDir string) bool {
	if dataDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dataDir, "session.json"))
	return err == nil
}

// isDevkitMCPTool identifies tools that are part of the devkit MCP
// server's own surface — and therefore safe to allow during a command
// or prompt step because they drive the engine, not the agent.
//
// Iteration history:
//  1. Pre-#64: mcp__*devkit* — unanchored substring. Any third-party
//     tool name containing "devkit" would bypass (e.g.
//     mcp__plugin_evil__devkit_masquerade).
//  2. PR #64: mcp__*devkit-engine*|mcp__devkit__* — required the
//     server name somewhere in the string. Still loose enough that
//     a plugin with "devkit-engine" elsewhere in its tool name would
//     pass.
//  3. This PR: anchored on the full plugin+server prefix. Even a
//     future second MCP server under the devkit plugin cannot
//     silently inherit command-step permissions — it would have to
//     be added to this allowlist deliberately.
//
// Accepted forms:
//   - devkit_advance / devkit_status / devkit_list / devkit_start
//     (bare names used in local dev and direct stdio callers)
//   - mcp__plugin_devkit_devkit-engine__<tool>
//     (Claude Code plugin convention: plugin=devkit, server=devkit-engine)
//   - mcp__devkit__<tool>
//     (short-form namespace for non-plugin MCP hosts that register the
//     server directly)
//
// Any other mcp__* tool falls through to default-deny.
func isDevkitMCPTool(name string) bool {
	switch name {
	case "devkit_advance", "devkit_status", "devkit_list", "devkit_start":
		return true
	}
	if strings.HasPrefix(name, "mcp__plugin_devkit_devkit-engine__") {
		return true
	}
	if strings.HasPrefix(name, "mcp__devkit__") {
		return true
	}
	return false
}

// isCompanionRescueCommand reports whether a Bash command string is a
// single, direct invocation of a known rescue-subagent companion script.
// Used by the prompt+hard carve-out so tri-* workflows' rescue subagents
// (codex, gemini) can reach external models through their companion
// adapters.
//
// Accepted shapes (after shell-word parsing):
//
//	node <path>/codex-companion.mjs <args...>
//	node <path>/gemini-companion.mjs <args...>
//	bash|sh -c '<one of the above>'
//
// Anything that could run a second command is rejected: unquoted
// operators (; & | < > newline), comments, backticks, and $( ) command
// substitution outside single quotes. Arguments may contain those
// characters only when quoted, so a raw prompt like
// `task "review: a && b"` still works. A substring match is NOT enough:
// `touch x # codex-companion.mjs` would satisfy it and let the main
// agent run arbitrary Bash during a hard prompt step.
func isCompanionRescueCommand(cmd string) bool {
	return isCompanionInvocation(cmd, 0)
}

var paramTransform = regexp.MustCompile(`\$\{[^}]*@[A-Za-z]\}`)

var companionScripts = map[string]bool{
	"codex-companion.mjs":  true,
	"gemini-companion.mjs": true,
}

func isCompanionInvocation(cmd string, depth int) bool {
	if depth > 1 {
		return false
	}
	words, ok := splitSimpleCommand(cmd)
	if !ok || len(words) < 2 {
		return false
	}
	prog, script := words[0], words[1]
	// ${var@P} (prompt expansion) re-parses the value and runs any
	// $(...) inside it, even within double quotes. Reject every
	// ${...@op} transformation in any word.
	for _, w := range words {
		if w.anyExp && paramTransform.MatchString(w.text) {
			return false
		}
	}
	// An unquoted $ in the program or script word is subject to field
	// splitting: `node ${IFS}-e${IFS}'code'${IFS}/x/codex-companion.mjs`
	// parses as two words here but bash runs `node -e code ...`.
	// Quoted expansions ("${CLAUDE_PLUGIN_ROOT}/...") don't split.
	if prog.unquotedExp || script.unquotedExp {
		return false
	}
	// Quoted expansions can still rewrite the word: "${PWD:+--eval}..."
	// turns the script into a node option. The program word must be
	// literal, and the script word may only reference the plugin root
	// as a plain variable — no operators, no other variables.
	if prog.anyExp {
		return false
	}
	if script.anyExp && strings.Contains(pluginRootRef.ReplaceAllString(script.text, ""), "$") {
		return false
	}
	// `NAME=value cmd` makes the first word an env assignment, so the
	// real program is word 2 (e.g. NODE_OPTIONS='--import=...#/node').
	if strings.Contains(prog.text, "=") {
		return false
	}
	switch strings.TrimSuffix(baseName(prog.text), ".exe") {
	case "node":
		// A leading - makes the "script" a node option:
		// node '--eval=code///codex-companion.mjs' runs code.
		if strings.HasPrefix(script.text, "-") || strings.Contains(script.text, "=") {
			return false
		}
		return companionScripts[baseName(script.text)]
	case "bash", "sh":
		// The -c script must reach the inner shell exactly as written:
		// any $ the OUTER shell expands (unquoted, double-quoted, or
		// $'...' ANSI-C quoting) can inject separators after this check.
		if len(words) == 3 && script.text == "-c" && !words[2].anyExp {
			return isCompanionInvocation(words[2].text, depth+1)
		}
	}
	return false
}

// pluginRootRef matches the only expansion allowed in a companion script
// path: a plain reference to CLAUDE_PLUGIN_ROOT.
var pluginRootRef = regexp.MustCompile(`\$\{CLAUDE_PLUGIN_ROOT\}|\$CLAUDE_PLUGIN_ROOT\b`)

// shellWord is one word of a simple command after quote removal.
type shellWord struct {
	text        string
	unquotedExp bool // contains a $ outside any quotes (field-split by the shell)
	anyExp      bool // contains a $ the shell would expand (unquoted or in double quotes)
}

// baseName returns the last path segment, splitting on both / and \\ so
// Windows-style script paths resolve the same way as POSIX ones.
func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// splitSimpleCommand splits cmd into shell words, honouring single
// quotes, double quotes and backslash escapes. It returns ok=false if
// the string is anything other than one simple command: unquoted
// control/redirect operators, comments, newlines, backticks, or $( )
// outside single quotes all fail. Variable references like
// ${CLAUDE_PLUGIN_ROOT} are left unexpanded in the returned words, with
// flags recording where expansions occur.
// paramRefEnd checks the expansion starting at cmd[i] == '$' and returns
// the index just past it. Only plain references are accepted: $NAME,
// ${NAME}, or a lone $. Anything inside ${...} beyond a name — subscripts
// (${x['$(cmd)']} runs cmd), operators, transformations — and $[...]
// arithmetic can execute code, so it is rejected.
func paramRefEnd(cmd string, i int) (int, bool) {
	j := i + 1
	if j >= len(cmd) {
		return j, true
	}
	switch cmd[j] {
	case '[':
		return 0, false
	case '{':
		k := j + 1
		for k < len(cmd) && isNameByte(cmd[k], k == j+1) {
			k++
		}
		if k == j+1 || k >= len(cmd) || cmd[k] != '}' {
			return 0, false
		}
		return k + 1, true
	}
	return j, true
}

func isNameByte(b byte, first bool) bool {
	switch {
	case b == '_', b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z':
		return true
	case b >= '0' && b <= '9':
		return !first
	}
	return false
}

func splitSimpleCommand(cmd string) ([]shellWord, bool) {
	var words []shellWord
	var cur strings.Builder
	var w shellWord
	inWord := false
	flush := func() {
		w.text = cur.String()
		words = append(words, w)
		cur.Reset()
		w = shellWord{}
		inWord = false
	}
	const (
		none = iota
		single
		double
	)
	quote := none
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch quote {
		case single:
			if c == '\'' {
				quote = none
			} else {
				cur.WriteByte(c)
			}
			continue
		case double:
			switch {
			case c == '"':
				quote = none
			case c == '`':
				return nil, false
			case c == '$' && i+1 < len(cmd) && cmd[i+1] == '(':
				return nil, false
			case c == '$':
				end, ok := paramRefEnd(cmd, i)
				if !ok {
					return nil, false
				}
				w.anyExp = true
				cur.WriteString(cmd[i:end])
				i = end - 1
			case c == '\\' && i+1 < len(cmd):
				// Inside double quotes a backslash only escapes $ ` " \
				// and newline; before anything else bash keeps it.
				switch cmd[i+1] {
				case '$', '`', '"', '\\':
					i++
					cur.WriteByte(cmd[i])
				case '\n':
					i++
				default:
					cur.WriteByte(c)
				}
			default:
				cur.WriteByte(c)
			}
			continue
		}
		switch c {
		case ' ', '\t':
			if inWord {
				flush()
			}
		case '\'':
			quote, inWord = single, true
		case '"':
			quote, inWord = double, true
		case '\\':
			if i+1 >= len(cmd) || cmd[i+1] == '\n' {
				return nil, false
			}
			i++
			cur.WriteByte(cmd[i])
			inWord = true
		case ';', '&', '|', '<', '>', '\n', '\r', '`', '(', ')':
			return nil, false
		case '#':
			if !inWord {
				return nil, false
			}
			cur.WriteByte(c)
		case '$':
			if i+1 < len(cmd) && cmd[i+1] == '(' {
				return nil, false
			}
			end, ok := paramRefEnd(cmd, i)
			if !ok {
				return nil, false
			}
			w.unquotedExp, w.anyExp = true, true
			cur.WriteString(cmd[i:end])
			i = end - 1
			inWord = true
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if quote != none {
		return nil, false
	}
	if inWord {
		flush()
	}
	return words, true
}

func runPreToolGuard() {
	dataDir := resolveDataDir(false)
	if dataDir == "" {
		guardExit(0)
		return
	}

	// Hot path: no active workflow → single stat, zero writes. This
	// must come BEFORE lib.ReadSessionJSON so we don't trigger the
	// withSessionLock mkdir + lock-file create side effects for users
	// who have $CLAUDE_PLUGIN_DATA set but no running session.
	if !sessionFileExists(dataDir) {
		guardExit(0)
		return
	}

	state, err := lib.ReadSessionJSON(dataDir)
	if err != nil {
		// Any error from ReadSessionJSON after sessionFileExists
		// returned true is a real problem: parse failure, lock
		// acquire failure, permission denial, quota exceeded, or a
		// filesystem error. Fail closed unconditionally — silently
		// allowing on permission errors would let a corrupted plugin
		// data dir disarm the guard without any user-visible signal.
		// The "no session" case is signalled by (nil, nil) from
		// ReadSessionJSON and handled below, so we never reach here
		// on a legitimately-absent session.
		sessionPath := filepath.Join(dataDir, "session.json")
		fmt.Fprintf(guardStderr,
			"BLOCKED: devkit-guard cannot read session state at %s. "+
				"Check permissions and JSON validity. Remove the file to clear. (%v)\n",
			sessionPath, err)
		guardExit(2)
		return
	}
	if state == nil || state.Status != "running" {
		// state == nil handles the TOCTOU where the file existed at
		// stat time but was removed before ReadSessionJSON acquired
		// the lock — ReadSessionJSON returns (nil, nil) for os.IsNotExist.
		guardExit(0)
		return
	}

	if sessionIsStale(state) {
		fmt.Fprintf(guardStderr,
			"devkit-guard: session %s idle past TTL — treating as orphaned (run devkit_start to reclaim)\n",
			state.Workflow)
		guardExit(0)
		return
	}

	// Cross-repo scope (mirrors stop-guard carve-out from issue #91):
	// if the workflow was started in a different repo than the current
	// Claude Code session, allow silently. Without this, a workflow
	// stuck at a prompt step in repo A (crashed engine, missed
	// devkit_advance, session not cleaned up) would keep blocking
	// unrelated tool calls in repo B for the full 30-minute TTL. The
	// block remains active in the originating repo — re-entering A
	// re-arms it. Same posture as stop-guard: empty state.RepoRoot
	// (pre-#91 sessions written by older engines) or unresolvable
	// current repo (CLAUDE_PROJECT_DIR missing, no walk-up match)
	// falls through to full enforcement so a missing signal never
	// silently approves.
	if state.RepoRoot != "" {
		if cur := currentRepoRoot(); cur != "" && !samePath(state.RepoRoot, cur) {
			fmt.Fprintf(guardStderr,
				"devkit-guard: session %s belongs to %s, current repo is %s — allowing (block remains active in originating repo)\n",
				state.Workflow, state.RepoRoot, cur)
			guardExit(0)
			return
		}
	}

	tool := guardToolName
	var command string
	if tool == "" {
		t, c, terr := readToolInvocationFromStdin()
		if terr != nil {
			// Log distinctly so schema drift in the PreToolUse payload
			// doesn't silently degrade into "unknown tool name" and
			// fall through to default-deny (or worse, default-allow
			// under parallel). The empty tool name we fall through
			// with still blocks under hard enforcement.
			fmt.Fprintf(guardStderr,
				"devkit-guard: cannot determine tool name from stdin: %v — treating as unknown\n",
				terr)
		}
		tool = t
		command = c
	}
	// state.StepEnforce is guaranteed valid ("hard" or "soft") by
	// SessionState.UnmarshalJSON — ReadSessionJSON would have rejected
	// a stale/corrupt session with a missing or invalid enforce field.
	enforce := state.StepEnforce

	// attemptedTool is what we print in veto diagnostics. An empty
	// tool name means we failed to parse it from stdin (or got an
	// intentional "" from a flag default); surface that explicitly
	// so users don't see a dangling "attempted tool: " paren.
	attemptedTool := tool
	if attemptedTool == "" {
		attemptedTool = "<unknown>"
	}

	switch state.StepType {
	case "command":
		if enforce != lib.EnforceHard {
			guardExit(0)
			return
		}
		if isDevkitMCPTool(tool) || tool == "TodoWrite" || tool == "Skill" {
			guardExit(0)
			return
		}
		fmt.Fprintf(guardStderr,
			"BLOCKED: Command step %q in progress — the engine runs this step. Call devkit_advance to execute it. (attempted tool: %s)\n",
			stepLabel(state), attemptedTool)
		guardExit(2)
		return

	case "prompt", "parallel":
		// A parallel dispatcher is a prompt step whose work runs in
		// subagents; it carries the same fence (Agent/Task allowed,
		// Write/Edit/Bash blocked under hard). MCP parks on the
		// dispatcher now that children are skipped in the walk.
		if enforce == lib.EnforceHard {
			if isDevkitMCPTool(tool) {
				guardExit(0)
				return
			}
			switch tool {
			case "Read", "Grep", "Glob", "TodoWrite", "NotebookRead", "Skill":
				guardExit(0)
				return
			case "Agent", "Task":
				// Subagent dispatch. The main agent hands off to a
				// subagent whose tool list is gated independently —
				// Write/Edit/Bash remain blocked at this layer, so the
				// main model cannot cheat (e.g. fake a tri-review by
				// writing the verdict itself instead of dispatching to
				// an external reviewer). Subagent tool calls re-enter
				// this guard with the same session state, so the fence
				// applies uniformly to both layers.
				guardExit(0)
				return
			case "Bash":
				// Rescue-subagent escape hatch (issue #95-2). tri-*
				// workflows dispatch codex:codex-rescue /
				// gemini:gemini-rescue via Agent/Task, and those
				// subagents' system prompts require forwarding to
				// codex-companion.mjs / gemini-companion.mjs through
				// Bash. That nested Bash call re-enters this guard
				// with the same prompt+hard session state and would
				// otherwise be blocked, silently breaking tri-review's
				// model-diversity invariant.
				//
				// The main agent could technically invoke these
				// companion scripts directly to fish for a sympathetic
				// external review, but that just runs the real
				// external model — there's no verdict-faking bypass
				// to be had. Write/Edit stay blocked, so the main
				// model still cannot author the review itself.
				if isCompanionRescueCommand(command) {
					guardExit(0)
					return
				}
			}
			fmt.Fprintf(guardStderr,
				"BLOCKED: devkit workflow %s is at a prompt step — gather evidence with Read/Grep/Glob then call devkit_advance. (attempted tool: %s)\n",
				stepLabel(state), attemptedTool)
			if attemptedTool == "Bash" && strings.Contains(command, "companion.mjs") {
				fmt.Fprintf(guardStderr,
					"hint: companion calls are allowed only as one plain command — node \"<path>/codex-companion.mjs\" task '<prompt>' — with plain quotes and no $( ), heredocs, pipes, redirects, or chained commands.\n")
			}
			guardExit(2)
			return
		}
		// Soft enforcement: allow + nudge. Idempotent — the Stop-hook
		// still blocks if the agent rides the nudge indefinitely.
		fmt.Fprintf(guardStderr,
			"devkit-guard: %s is open — call devkit_advance when the step is complete.\n",
			stepLabel(state))
		guardExit(0)
		return

	default:
		// unknown step type — allow through rather than wedge the session.
		guardExit(0)
	}
}

// stopVerdict is the JSON shape consumed by Claude Code's Stop hook.
type stopVerdict struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

// writeStopVerdict emits the JSON verdict on stdout with no trailing
// newline, matching the shell hook's exact byte output. json.Marshal
// on a plain string-only struct cannot fail, so we do not carry a
// fallback branch — if this ever panics the test suite will catch it.
func writeStopVerdict(v stopVerdict) {
	data, err := json.Marshal(v)
	if err != nil {
		// Truly unreachable for stopVerdict's shape (only strings).
		// If the struct ever grows a field that can fail to marshal,
		// we want the panic in test/CI rather than silently emitting
		// a hardcoded block verdict that hides the bug.
		panic(fmt.Errorf("devkit-stop-guard: unreachable json.Marshal failure on stopVerdict: %w", err))
	}
	if _, err := fmt.Fprint(guardStdout, string(data)); err != nil {
		// Stdout is the Stop hook's only channel. Log to stderr so a
		// broken pipe leaves some post-mortem trail, but still return
		// cleanly so Claude Code doesn't time out waiting for us.
		fmt.Fprintf(guardStderr,
			"devkit-stop-guard: failed to write verdict to stdout: %v\n", err)
	}
}

func runStopGuard() {
	dataDir := resolveDataDir(true)
	if dataDir == "" {
		writeStopVerdict(stopVerdict{Decision: "approve"})
		guardExit(0)
		return
	}

	// Hot path: no session file → approve. Same reason as the Pre
	// variant — avoid the mkdir/lock side effects on the cold path.
	if !sessionFileExists(dataDir) {
		writeStopVerdict(stopVerdict{Decision: "approve"})
		guardExit(0)
		return
	}

	state, err := lib.ReadSessionJSON(dataDir)
	if err != nil {
		// Any error after a positive sessionFileExists is a real
		// problem — fail closed with an actionable reason. The Stop
		// hook's fail-closed verdict is a BLOCK so the user sees the
		// diagnostic in their transcript and can recover manually
		// rather than shipping a half-finished workflow.
		sessionPath := filepath.Join(dataDir, "session.json")
		fmt.Fprintf(guardStderr,
			"devkit-stop-guard: cannot read session state at %s: %v — blocking Stop\n",
			sessionPath, err)
		writeStopVerdict(stopVerdict{
			Decision: "block",
			Reason: fmt.Sprintf(
				"devkit session state unreadable — remove %s to clear", sessionPath),
		})
		guardExit(0)
		return
	}
	if state == nil || state.Status != "running" {
		// state == nil handles the TOCTOU where the file was removed
		// between sessionFileExists and ReadSessionJSON.
		// Stop is enforce-agnostic — any running workflow blocks Stop
		// regardless of soft/hard — so we don't branch on state.StepEnforce.
		writeStopVerdict(stopVerdict{Decision: "approve"})
		guardExit(0)
		return
	}

	if sessionIsStale(state) {
		fmt.Fprintf(guardStderr,
			"devkit-stop-guard: session %s idle past TTL — approving Stop (reclaim on next devkit_start)\n",
			state.Workflow)
		writeStopVerdict(stopVerdict{Decision: "approve"})
		guardExit(0)
		return
	}

	// Scope restriction (issue #91): if the workflow was started in a
	// different repo than the one the current Claude Code session is
	// operating in, approve silently. The block is not a bypass — it
	// remains active when the user returns to the originating repo;
	// this only prevents a stuck workflow in repo A from nagging every
	// turn of unrelated work in repo B. No TTL escape, no env override,
	// no branch carve-out: if the session started in repo A, only repo A
	// sees the block. Empty state.RepoRoot (pre-#91 sessions) or empty
	// current repo root (hook env missing) falls through to the legacy
	// block behavior — we do not silently approve on missing signals.
	if state.RepoRoot != "" {
		if cur := currentRepoRoot(); cur != "" && !samePath(state.RepoRoot, cur) {
			fmt.Fprintf(guardStderr,
				"devkit-stop-guard: session %s belongs to %s, current repo is %s — approving Stop (block remains active in originating repo)\n",
				state.Workflow, state.RepoRoot, cur)
			writeStopVerdict(stopVerdict{Decision: "approve"})
			guardExit(0)
			return
		}
	}

	remaining := 0
	if state.TotalSteps > 0 {
		remaining = state.TotalSteps - state.CurrentIndex
		if remaining < 0 {
			remaining = 0
		}
	}
	wf := state.Workflow
	if wf == "" {
		wf = "unknown"
	}
	writeStopVerdict(stopVerdict{
		Decision: "block",
		Reason: fmt.Sprintf(
			"Workflow %s incomplete — %d steps remaining. Call devkit_advance to continue.",
			wf, remaining),
	})
	guardExit(0)
}
