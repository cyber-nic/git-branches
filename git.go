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
	switch branches[selected] {
	case branchMaster, branchMain:
		return nil
	default:
		branchToDelete = branches[selected]
		confirming = true
		return nil
	}
}

func confirmDelete(g *gocui.Gui, v *gocui.View) error {
	if !confirming {
		return nil // No confirmation needed
	}
	exec.Command("git", "branch", "-D", branchToDelete).Run()
	branches = getLocalBranches()
	selected = 0 // Reset selection after deletion
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
