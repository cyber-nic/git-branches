package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

type SortType int
type SortDirection int

const (
	SortAlphabetical SortType = iota
	SortCreationDate
	SortCommitDate
)

const (
	SortAscending SortDirection = iota
	SortDescending
)

type SortState struct {
	Type      SortType
	Direction SortDirection
}

var (
	gui            *gocui.Gui
	branches       []string
	defaultBranch  string
	sortState      = SortState{Type: SortAlphabetical, Direction: SortAscending}
	confirming     bool
	branchToDelete string
)

func main() {
	defaultBranch = "main"
	if !branchExists(defaultBranch) {
		defaultBranch = "master"
	}

	branches = getLocalBranches()
	// one‑time initial sort (no gui yet)
	sortBranches()

	var err error
	gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer gui.Close()

	gui.SetManagerFunc(layout)
	bindKeys()
	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Fprintln(os.Stderr, err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("branches", 0, 0, maxX-1, maxY-3); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == gocui.ErrUnknownView {
		v.Title = " branches "
	}

	if v, err := g.SetView("help", 0, maxY-3, maxX-1, maxY-1); err != nil && err != gocui.ErrUnknownView {
		return err
	} else if err == gocui.ErrUnknownView {
		fmt.Fprintf(v, "sort  [i] Index  [a] Alphabetical  [c] Creation  [u] Commit  [r] Reverse")
		fmt.Fprintf(v, "[d] Delete  [q] Quit")
	}

	if confirming {
		cx, cy := maxX/4, maxY/3
		if v, err := g.SetView("confirm", cx, cy, cx*3, cy+4); err != nil && err != gocui.ErrUnknownView {
			return err
		} else if err == gocui.ErrUnknownView {
			v.Title = " confirm "
			fmt.Fprintf(v, "Delete branch %q? (y/n)", branchToDelete)
		}
	} else {
		g.DeleteView("confirm")
	}

	return refreshBranchesView(g)
}

func refreshBranchesView(g *gocui.Gui) error {
	v, _ := g.View("branches")
	v.Clear()
	fmt.Fprintf(v, "%-4s %-20s %-16s %s\n", "#", "Branch", "Last Commit", "Ahead/Behind")
	for i, br := range branches {
		lt := getLastCommitTime(br).Format("2006-01-02 15:04")
		a, b := getAheadBehind(defaultBranch, br)
		fmt.Fprintf(v, "%-4d %-20s %-16s %d/%d\n", i+1, br, lt, a, b)
	}
	return nil
}

func bindKeys() {
	gui.SetKeybinding("", 'q', gocui.ModNone, func(*gocui.Gui, *gocui.View) error { return gocui.ErrQuit })
	gui.SetKeybinding("", 'a', gocui.ModNone, makeSorter(SortAlphabetical))
	gui.SetKeybinding("", 'c', gocui.ModNone, makeSorter(SortCreationDate))
	gui.SetKeybinding("", 'u', gocui.ModNone, makeSorter(SortCommitDate))
	gui.SetKeybinding("", 'r', gocui.ModNone, toggleDirection)
	gui.SetKeybinding("", 'd', gocui.ModNone, promptDelete)
	gui.SetKeybinding("confirm", 'y', gocui.ModNone, confirmDelete)
	gui.SetKeybinding("confirm", 'n', gocui.ModNone, cancelDelete)
}

func makeSorter(t SortType) func(*gocui.Gui, *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		sortState.Type = t
		sortBranches()
		return nil
	}
}

func toggleDirection(g *gocui.Gui, v *gocui.View) error {
	if sortState.Direction == SortAscending {
		sortState.Direction = SortDescending
	} else {
		sortState.Direction = SortAscending
	}
	sortBranches()
	return nil
}

// shared sort logic; updates view via gui.Update if ready
func sortBranches() {
	switch sortState.Type {
	case SortAlphabetical:
		if sortState.Direction == SortAscending {
			sort.Strings(branches)
		} else {
			sort.Sort(sort.Reverse(sort.StringSlice(branches)))
		}
	case SortCreationDate:
		sort.Slice(branches, func(i, j int) bool {
			t1 := getBranchCreationTime(branches[i])
			t2 := getBranchCreationTime(branches[j])
			if sortState.Direction == SortAscending {
				return t1.Before(t2)
			}
			return t1.After(t2)
		})
	case SortCommitDate:
		sort.Slice(branches, func(i, j int) bool {
			t1 := getLastCommitTime(branches[i])
			t2 := getLastCommitTime(branches[j])
			if sortState.Direction == SortAscending {
				return t1.Before(t2)
			}
			return t1.After(t2)
		})
	}
	if gui != nil {
		gui.Update(func(g *gocui.Gui) error { return refreshBranchesView(g) })
	}
}

func promptDelete(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	// subtract 1 for header
	if idx := cy - 1; idx >= 0 && idx < len(branches) {
		branchToDelete = branches[idx]
		confirming = true
	}
	return nil
}

func confirmDelete(g *gocui.Gui, v *gocui.View) error {
	exec.Command("git", "branch", "-D", branchToDelete).Run()
	branches = getLocalBranches()
	sortBranches()
	confirming = false
	return nil
}

func cancelDelete(g *gocui.Gui, v *gocui.View) error {
	confirming = false
	return nil
}

func branchExists(name string) bool {
	return exec.Command("git", "rev-parse", "--verify", name).Run() == nil
}

func getLocalBranches() []string {
	out, err := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads/").Output()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

func getLastCommitTime(branch string) time.Time {
	out, err := exec.Command("git", "log", "-1", "--format=%ct", branch).Output()
	if err != nil {
		return time.Time{}
	}
	sec, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return time.Unix(sec, 0)
}

func getBranchCreationTime(branch string) time.Time {
	out, err := exec.Command("git", "log", "--format=%ct", "--reverse", branch, "-n", "1").Output()
	if err != nil {
		return time.Time{}
	}
	sec, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return time.Unix(sec, 0)
}

func getAheadBehind(base, branch string) (int, int) {
	out, err := exec.Command("git", "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", base, branch)).Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 0, 0
	}
	behind, _ := strconv.Atoi(parts[0])
	ahead, _ := strconv.Atoi(parts[1])
	return ahead, behind
}
