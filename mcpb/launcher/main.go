// devkit MCPB Windows launcher — wiring probe stub.
//
// This is a PROVING-GROUND build, not the real launcher. Its only job is to
// confirm, on a real Windows box in Claude Code's MCP spawn context, that:
//
//   1. plugin.json's mcpServers path → MCPB unpack works on Windows
//   2. platform_overrides.win32 → the win32 entry fires (not the base entry)
//   3. ${__dirname} → expands to a CreateProcess-compatible Windows path
//   4. A Go-compiled PE binary under .mcpb-cache/<hash>/server/ actually
//      executes when CC spawns it
//   5. CLAUDE_PLUGIN_ROOT and CLAUDE_PLUGIN_DATA env vars are exported to
//      the spawned MCP child
//
// Everything gets logged to stderr so it lands in CC's MCP error log.
// There's no JSON-RPC handshake — /mcp will show this as failed. That's
// fine; we're verifying the spawn layer, not the protocol layer. Once this
// is verified, a follow-up commit replaces this stub with the real
// download + checksum + exec launcher that ports bin/devkit to Go.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

func main() {
	// Marker string — must be distinctive so the reporter can grep for it
	// in CC's MCP error output and confirm this exact build ran.
	fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: WIN32_OVERRIDE_ACTIVE")
	fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: runtime.GOOS="+runtime.GOOS+" GOARCH="+runtime.GOARCH)

	exe, _ := os.Executable()
	fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: executable="+exe)

	wd, _ := os.Getwd()
	fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: cwd="+wd)

	fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: args="+strings.Join(os.Args, " "))

	for _, name := range []string{"CLAUDE_PLUGIN_ROOT", "CLAUDE_PLUGIN_DATA", "PATH"} {
		val := os.Getenv(name)
		if name == "PATH" && len(val) > 200 {
			val = val[:200] + "...(truncated)"
		}
		fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: "+name+"="+val)
	}

	fmt.Fprintln(os.Stderr, "DEVKIT-WIRING-PROBE: sleeping 60s then exiting 0")
	time.Sleep(60 * time.Second)
}
