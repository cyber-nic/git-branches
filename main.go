package main

import (
	"fmt"
	"log"

	"github.com/jroimartin/gocui"
)

var (
	defaultBranch  = "main"
	branches       []string
	branchToDelete string
	confirming     bool
	selected       = 0
)

func main() {
	if !branchExists(defaultBranch) {
		defaultBranch = "master"
	}

	branches = getLocalBranches()

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
	if err := g.SetKeybinding("", 'q', gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return gocui.ErrQuit }); err != nil {
		return fmt.Errorf("failed to set keybinding: %v", err)
	}
	// gui.SetKeybinding("", 'a', gocui.ModNone, makeSorter(SortAlphabetical))
	// gui.SetKeybinding("", 'c', gocui.ModNone, makeSorter(SortCreationDate))
	// gui.SetKeybinding("", 'u', gocui.ModNone, makeSorter(SortCommitDate))
	// gui.SetKeybinding("", 'r', gocui.ModNone, toggleDirection)
	g.SetKeybinding("", 'd', gocui.ModNone, promptDelete)
	g.SetKeybinding(viewDeleteBranch, 'y', gocui.ModNone, confirmDelete)
	g.SetKeybinding(viewDeleteBranch, 'n', gocui.ModNone, cancelDelete)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return fmt.Errorf("failed to set keybinding: %v", err)
	}

	if err := g.SetKeybinding(viewBranches, gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return fmt.Errorf("failed to set keybinding: %v", err)
	}

	if err := g.SetKeybinding(viewBranches, gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return fmt.Errorf("failed to set keybinding: %v", err)
	}

	if err := g.SetKeybinding(viewBranches, gocui.MouseLeft, gocui.ModNone, mouseClick); err != nil {
		return fmt.Errorf("failed to set keybinding: %v", err)
	}
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if confirming {
		cx, cy := maxX/4, maxY/3
		if v, err := g.SetView(viewDeleteBranch, cx, cy, cx*3, cy+4); err != nil && err != gocui.ErrUnknownView {
			return err
		} else if err == gocui.ErrUnknownView {
			v.Title = " confirm "
			fmt.Fprintf(v, "Delete branch %q? (y/n)", branchToDelete)
		}

		if _, err := g.SetCurrentView(viewDeleteBranch); err != nil {
			return fmt.Errorf("failed to set current view: %w", err)
		}
	} else {
		g.DeleteView(viewDeleteBranch)
	}

	// find the longest branch name for formatting
	maxBranchLen := 64
	for _, br := range branches {
		if len(br) > maxBranchLen {
			maxBranchLen = len(br)
		}
	}
	// Adjust the header to match the longest branch name
	// lineFormat := fmt.Sprintf("%-4s %-*s %16s %9s", maxBranchLen)
	lineFormat := fmt.Sprintf("%%-4d %%-%ds %%32s %%20s", maxBranchLen)
	lineTitle := fmt.Sprintf("%-4s %-*s %32s %20s\n", "#", maxBranchLen, "Branch", "Last Commit", "Ahead/Behind")

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
	// header

	// Render each color, prefixing the selected one with an arrow
	for i, b := range branches {
		ahead, behind := getAheadBehind(defaultBranch, b)
		lastCom := getLastCommitTime(b).Format("2006-01-02 15:04")

		line := fmt.Sprintf(lineFormat, i+1, b, lastCom, fmt.Sprintf("%d/%d", ahead, behind))

		if i == selected {
			fmt.Fprintf(v, "\033[30;42m➜ %s\033[0m\n", line)
			continue
		}
		fmt.Fprintf(v, "  %s\n", line)
	}

	if v, err := g.SetView("help", 0, maxY-3, maxX-1, maxY-1); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == gocui.ErrUnknownView {
		fmt.Fprintf(v, "sort  [i] Index  [a] Alphabetical  [c] Creation  [u] Commit  [r] Reverse")
		fmt.Fprintf(v, "[d] Delete  [q] Quit")
	}

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
