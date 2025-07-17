package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

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

// func getBranchCreationTime(branch string) time.Time {
// 	out, err := exec.Command("git", "log", "--format=%ct", "--reverse", branch, "-n", "1").Output()
// 	if err != nil {
// 		return time.Time{}
// 	}
// 	sec, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
// 	return time.Unix(sec, 0)
// }

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

func promptDelete(g *gocui.Gui, v *gocui.View) error {
	if branches[selected] == "master" || branches[selected] == "main" {
		return nil
	}
	branchToDelete = branches[selected]
	confirming = true
	return nil
}

func confirmDelete(g *gocui.Gui, v *gocui.View) error {
	if !confirming {
		return nil // No confirmation needed
	}
	exec.Command("git", "branch", "-D", branchToDelete).Run()
	branches = getLocalBranches()
	selected = 0 // Reset selection after deletion
	// sortBranches()
	confirming = false
	return nil
}

func cancelDelete(g *gocui.Gui, v *gocui.View) error {
	confirming = false
	return nil
}

// func checkoutBranch
func checkoutBranch(_ *gocui.Gui, _ *gocui.View) error {
	if selected < 0 || selected >= len(branches) {
		return nil // No valid selection
	}
	branch := branches[selected]
	if branchExists(branch) {
		if err := exec.Command("git", "checkout", branch).Run(); err != nil {
			return fmt.Errorf("failed to checkout branch %s: %v", branch, err)
		}
		return gocui.ErrQuit // Exit after checkout
	}
	return fmt.Errorf("branch %s does not exist", branch)
}

// // func makeSorter(t SortType) func(*gocui.Gui, *gocui.View) error {
// // 	return func(g *gocui.Gui, v *gocui.View) error {
// // 		sortState.Type = t
// // 		sortBranches()
// // 		return nil
// // 	}
// // }

// // func toggleDirection(g *gocui.Gui, v *gocui.View) error {
// // 	if sortState.Direction == SortAscending {
// // 		sortState.Direction = SortDescending
// // 	} else {
// // 		sortState.Direction = SortAscending
// // 	}
// // 	sortBranches()
// // 	return nil
// // }

// // shared sort logic; updates view via gui.Update if ready
// func sortBranches(g *gocui.Gui) {
// 	switch sortState.Type {
// 	case SortAlphabetical:
// 		if sortState.Direction == SortAscending {
// 			sort.Strings(branches)
// 		} else {
// 			sort.Sort(sort.Reverse(sort.StringSlice(branches)))
// 		}
// 	case SortCreationDate:
// 		sort.Slice(branches, func(i, j int) bool {
// 			t1 := getBranchCreationTime(branches[i])
// 			t2 := getBranchCreationTime(branches[j])
// 			if sortState.Direction == SortAscending {
// 				return t1.Before(t2)
// 			}
// 			return t1.After(t2)
// 		})
// 	case SortCommitDate:
// 		sort.Slice(branches, func(i, j int) bool {
// 			t1 := getLastCommitTime(branches[i])
// 			t2 := getLastCommitTime(branches[j])
// 			if sortState.Direction == SortAscending {
// 				return t1.Before(t2)
// 			}
// 			return t1.After(t2)
// 		})
// 	}
// 	if g != nil {
// 		g.Update(func(gui *gocui.Gui) error { return refreshBranchesView(gui) })
// 	}
// }

// func refreshBranchesView(g *gocui.Gui) error {
// 	v, _ := g.View(string(Branches))
// 	v.Clear()
// 	fmt.Fprintf(v, "%-4s %-20s %16s %9s\n", "#", "Branch", "Last Commit", "Ahead/Behind")
// 	for i, br := range branches {
// 		lt := getLastCommitTime(br).Format("2006-01-02 15:04")
// 		a, b := getAheadBehind(defaultBranch, br)
// 		fmt.Fprintf(v, "%-4d %-20s %16s %9s\n",
// 			i+1, br, lt, fmt.Sprintf("%d/%d", a, b),
// 		)
// 	}
// 	// // keep selectedIdx in range
// 	// if selectedIdx >= len(branches) {
// 	// 	selectedIdx = len(branches) - 1
// 	// }
// 	// // highlight the “cursor” (header is row 0, so data starts at row 1)
// 	// v.SetCursor(0, selectedIdx+1)

//		return nil
//	}
