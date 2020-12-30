package osdetail

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
	"time"

	"github.com/safing/portbase/log"

	"github.com/safing/portbase/info"
	"github.com/safing/portbase/modules"
	"github.com/shirou/gopsutil/host"
)

// DebugInfo gathers debugging information and stores everything in a buffer in
// order to write it to somewhere later. It directly inherits a bytes.Buffer,
// so you can also use all these functions too.
type DebugInfo struct {
	bytes.Buffer
	Style string
}

// DebugInfoFlag defines possible options for adding sections to a DebugInfo.
type DebugInfoFlag int

const (
	// NoFlags does nothing.
	NoFlags DebugInfoFlag = 0

	// UseCodeSection wraps the section content in a markdown code section.
	UseCodeSection DebugInfoFlag = 1

	// AddContentLineBreaks adds a line breaks after each line of content,
	// except for the last.
	AddContentLineBreaks DebugInfoFlag = 2
)

func useCodeSection(flags DebugInfoFlag) bool {
	return flags&UseCodeSection > 0
}

func addContentLineBreaks(flags DebugInfoFlag) bool {
	return flags&AddContentLineBreaks > 0
}

// AddSection adds a debug section to the DebugInfo. The result is directly
// written into the buffer.
func (di *DebugInfo) AddSection(name string, flags DebugInfoFlag, content ...string) {
	// Check if we need a spacer.
	if di.Len() > 0 {
		di.WriteString("\n\n")
	}

	// Write section to buffer.

	// Write section header.
	if di.Style == "github" {
		di.WriteString(fmt.Sprintf("<details>\n<summary>%s</summary>\n\n", name))
	} else {
		di.WriteString(fmt.Sprintf("**%s**:\n\n", name))
	}

	// Write section content.
	if useCodeSection(flags) {
		// Write code header: Needs one empty line between previous data.
		di.WriteString("```\n")
	}
	for i, part := range content {
		di.WriteString(part)
		if addContentLineBreaks(flags) && i < len(content)-1 {
			di.WriteString("\n")
		}
	}
	if useCodeSection(flags) {
		// Write code footer: Needs one empty line between next data.
		di.WriteString("\n```\n")
	}

	// Write section header.
	if di.Style == "github" {
		di.WriteString("\n</details>")
	}
}

// AddVersionInfo adds version information from the info pkg.
func (di *DebugInfo) AddVersionInfo() {
	di.AddSection(
		"Version "+info.Version(),
		UseCodeSection,
		info.FullVersion(),
	)
}

// AddPlatformInfo adds OS and platform information.
func (di *DebugInfo) AddPlatformInfo(ctx context.Context) {
	// Get information from the system.
	info, err := host.InfoWithContext(ctx)
	if err != nil {
		di.AddSection(
			"Platform Information",
			NoFlags,
			fmt.Sprintf("Failed to get: %s", err),
		)
		return
	}

	// Check if we want to add virtulization information.
	var virtInfo string
	if info.VirtualizationRole == "guest" {
		if info.VirtualizationSystem != "" {
			virtInfo = fmt.Sprintf("VM: %s", info.VirtualizationSystem)
		} else {
			virtInfo = "VM: unidentified"
		}
	}

	// Add section.
	di.AddSection(
		fmt.Sprintf("Platform: %s %s", info.Platform, info.PlatformVersion),
		UseCodeSection|AddContentLineBreaks,
		fmt.Sprintf("System: %s %s (%s) %s", info.Platform, info.OS, info.PlatformFamily, info.PlatformVersion),
		fmt.Sprintf("Kernel: %s %s", info.KernelVersion, info.KernelArch),
		virtInfo,
	)
}

// AddGoroutineStack adds the current goroutine stack.
func (di *DebugInfo) AddGoroutineStack() {
	buf := new(bytes.Buffer)
	err := pprof.Lookup("goroutine").WriteTo(buf, 1)
	if err != nil {
		di.AddSection(
			"Goroutine Stack",
			NoFlags,
			fmt.Sprintf("Failed to get: %s", err),
		)
		return
	}

	// Add section.
	di.AddSection(
		"Goroutine Stack",
		UseCodeSection,
		buf.String(),
	)
}

// AddLastReportedModuleError adds the last reported module error, if one exists.
func (di *DebugInfo) AddLastReportedModuleError() {
	me := modules.GetLastReportedError()
	if me == nil {
		di.AddSection("No Module Error", NoFlags)
		return
	}

	di.AddSection(
		"Module Error",
		UseCodeSection,
		me.Format(),
	)
}

// AddLastUnexpectedLogs adds the last 10 unexpected log lines, if any.
func (di *DebugInfo) AddLastUnexpectedLogs() {
	lines := log.GetLastUnexpectedLogs()

	// Check if there is anything at all.
	if len(lines) == 0 {
		di.AddSection("No Unexpected Logs", NoFlags)
		return
	}

	di.AddSection(
		"Unexpected Logs",
		UseCodeSection|AddContentLineBreaks,
		append(
			lines,
			fmt.Sprintf("%s CURRENT TIME", time.Now().Format("060102 15:04:05.000")),
		)...,
	)
}
