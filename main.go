package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"gopkg.in/yaml.v2"
)

const (
	branchMain   = "main"
	branchMaster = "master"
)

var (
	defaultBranch   = branchMain
	branches        []string
	branchToDelete  string
	confirming      bool
	selected        = 0
	branchInfoCache map[string]branchInfo
	cacheMutex      sync.RWMutex
	knownBranches   map[string][]string
)

// branchInfo cache structures
type branchInfo struct {
	ahead          int
	behind         int
	lastCommitTime string
	tags           bool
	name           string // branch name with tags if any
}

func initialize() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if !branchExists(defaultBranch) {
		defaultBranch = branchMaster
	}

	knownBranches = readKnownBranches()
	branches = getLocalBranches()
	branchInfoCache = make(map[string]branchInfo, len(branches))
}

func main() {
	initialize()

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	// Enable mouse support
	g.Mouse = true

	g.SetManagerFunc(layout)
	if err := bindKeys(g); err != nil {
		log.Panicln(fmt.Errorf("failed to set keybinding: %v", err))

	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

const viewBranches = "list_branches"
const viewDeleteBranch = "delete_branch"

func bindKeys(g *gocui.Gui) error {

	// General keys
	for key, handler := range map[interface{}]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyCtrlC:  quit,
		'q':             quit,
		gocui.KeyDelete: promptDelete,
		'd':             promptDelete,
	} {
		if err := g.SetKeybinding("", key, gocui.ModNone, handler); err != nil {
			return fmt.Errorf("failed to set keybinding %v: %w", key, err)
		}
	}

	// Navigation keys for delete branch confirmation view
	for key, handler := range map[interface{}]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEsc:   cancelDelete,
		gocui.KeyEnter: confirmDelete,
		'y':            confirmDelete,
		'n':            cancelDelete,
	} {
		if err := g.SetKeybinding(viewDeleteBranch, key, gocui.ModNone, handler); err != nil {
			return fmt.Errorf("failed to set keybinding %v: %w", key, err)
		}
	}

	// Navigation keys
	for key, handler := range map[interface{}]func(*gocui.Gui, *gocui.View) error{
		'r':                  refreshBranches,
		'f':                  fetchBranch,
		'p':                  pullBranch,
		gocui.KeyArrowUp:     cursorUp,
		gocui.KeyArrowDown:   cursorDown,
		gocui.MouseLeft:      mouseClick,
		gocui.MouseWheelUp:   cursorUp,       // scroll wheel support
		gocui.MouseWheelDown: cursorDown,     // scroll wheel support
		gocui.KeyEnter:       checkoutBranch, // checkout selected branch and exit

		// todo: add more navigation keys if needed
		// gocui.KeyPgup: pageUp,   // page navigation
		// gocui.KeyPgdn: pageDown, // page navigation
		// gocui.KeyHome: navigateToFirst,
		// gocui.KeyEnd:  navigateToLast,

	} {
		if err := g.SetKeybinding(viewBranches, key, gocui.ModNone, handler); err != nil {
			return fmt.Errorf("failed to set keybinding %v: %w", key, err)
		}
	}

	return nil
}

func cacheBranchInfo(branch string) branchInfo {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	var data branchInfo

	if data, exists := branchInfoCache[branch]; exists {
		return data
	}

	name := branch
	for _, tag := range knownBranches[branch] {
		name += fmt.Sprintf(" [%s]", tag)
	}

	ahead, behind := getAheadBehind(defaultBranch, branch)
	data = branchInfo{
		ahead:          ahead,
		behind:         behind,
		lastCommitTime: getLastCommitTime(branch).Format("2006-01-02 15:04"),
		tags:           len(knownBranches[branch]) > 0,
		name:           name,
	}
	branchInfoCache[branch] = data
	return data
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if confirming {
		cx, cy := maxX/4, maxY/3
		if v, err := g.SetView(viewDeleteBranch, cx, cy, cx*3, cy+4); err != nil && err != gocui.ErrUnknownView {
			return err
		} else if err == gocui.ErrUnknownView {
			v.Title = " confirm "
			fmt.Fprintf(v, "Delete branch %q? (Y/n)", branchToDelete)
		}

		if _, err := g.SetCurrentView(viewDeleteBranch); err != nil {
			return fmt.Errorf("failed to set current view: %w", err)
		}
	} else {
		g.DeleteView(viewDeleteBranch)
		if len(g.Views()) > 0 {
			if _, err := g.SetCurrentView(viewBranches); err != nil {
				log.Println("Failed to set current view:", err)
				return fmt.Errorf("failed to set current view: %w", err)
			}
		}
	}
	// Calculate available width and allocate space for columns
	maxBranchLen := 0
	for _, br := range branches {
		if len(br) > maxBranchLen {
			maxBranchLen = len(br)
		}
	}

	// Fixed widths for other columns
	idxWidth := 4   // Column for index number
	dateWidth := 19 // Last commit date column
	statWidth := 12 // Ahead/Behind stats column

	// Calculate branch column width to fill remaining space
	branchWidth := maxX - idxWidth - dateWidth - statWidth - 5 // 5 for spacing/borders
	branchWidth = int(math.Max(float64(branchWidth), 10))      // Ensure minimum width

	// Create format strings that use the full width
	lineFormat := fmt.Sprintf("%%-%dd %%-%ds %%-%ds %%-%ds", idxWidth, branchWidth, dateWidth, statWidth)
	lineTitle := fmt.Sprintf("%-*s %-*s %-*s %-*s", idxWidth, "#", branchWidth, "Branch", dateWidth, "Last Commit", statWidth, "+/-")

	if v, err := g.SetView(viewBranches, 0, 0, maxX-1, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = lineTitle

		if _, err := g.SetCurrentView(viewBranches); err != nil {
			return err
		}
	}

	v, _ := g.View(viewBranches)
	v.Clear()

	// Render each branch, prefixing the selected one with an arrow
	for i, b := range branches {
		// build and cache branch info
		info := cacheBranchInfo(b)

		switch {
		case i == selected: // green background with black text
			line := fmt.Sprintf(lineFormat, i+1, info.name, info.lastCommitTime, fmt.Sprintf("%d/%d", info.ahead, info.behind))
			fmt.Fprintf(v, "%s\n", color.New(color.BgGreen, color.FgBlack).Sprint("➜ "+line))
		case info.tags: // dark yellow foreground
			line := fmt.Sprintf(lineFormat, i+1, info.name, info.lastCommitTime, fmt.Sprintf("%d/%d", info.ahead, info.behind))
			fmt.Fprintf(v, "  %s\n", color.New(color.FgYellow).Sprint(line))
		default: // default color
			line := fmt.Sprintf(lineFormat, i+1, info.name, info.lastCommitTime, fmt.Sprintf("%d/%d", info.ahead, info.behind))
			fmt.Fprintf(v, "  %s\n", line)
		}

	}

	if v, err := g.SetView("help", 0, maxY-3, maxX-1, maxY-1); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == gocui.ErrUnknownView {
		fmt.Fprintf(v, "[r] refresh  [p] pull [d] Delete  [q] Quit")
	}

	return nil
}

func refreshBranches(g *gocui.Gui, v *gocui.View) error {
	initialize() // Reinitialize to refresh branches and cache
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if selected > 0 {
		selected--
	}
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if selected < len(branches)-1 {
		selected++
	}
	return nil
}

func mouseClick(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}
	_, y := v.Cursor()
	if y >= 0 && y < len(branches) {
		selected = y
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// readKnownBranches reads the .known_branches.yml file and returns a map of branch names to labels.
// Returns an empty map if the file doesn't exist.
func readKnownBranches() map[string][]string {
	// Try to open the file
	file, err := os.Open(".known_branches.yml")
	if err != nil {
		// Return empty map if file doesn't exist
		return make(map[string][]string)
	}
	defer file.Close()

	// Decode the YAML file
	var branchLabels map[string][]string
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&branchLabels); err != nil {
		// If there's an error parsing, log it and return empty map
		log.Printf("Error parsing .known_branches.yml: %v", err)
		return make(map[string][]string)
	}

	return branchLabels
}
