package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
)

func main() {
	// Determine default branch
	defaultBranch := "main"
	if !branchExists(defaultBranch) {
		defaultBranch = "master"
	}

	renderTable(defaultBranch)
}

func renderTable(defaultBranch string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Branch", "Last Commit", "Ahead\\Behind", "Delete"})

	for _, br := range getLocalBranches() {
		commitTime := getLastCommitTime(br)
		timeStr := commitTime.Format("2006-01-02 15:04")

		ahead, behind := getAheadBehind(defaultBranch, br)
		track := fmt.Sprintf("%d/%d", ahead, behind)

		// 'delete' hyperlink triggers this app with args
		link := fmt.Sprintf("\x1b]8;;git-branches delete %s\x1b\\delete\x1b]8;;\x1b\\", br)

		t.AppendRow(table.Row{br, timeStr, track, link})
	}

	t.Render()
}

func branchExists(name string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", name)
	err := cmd.Run()
	return err == nil
}

func getLocalBranches() []string {
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

func getLastCommitTime(branch string) time.Time {
	cmd := exec.Command("git", "log", "-1", "--format=%ct", branch)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}
	sec, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return time.Unix(sec, 0)
}

func getAheadBehind(base, branch string) (int, int) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", fmt.Sprintf("%s...%s", base, branch))
	out, err := cmd.Output()
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
