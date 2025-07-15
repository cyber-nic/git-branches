package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Define sort types and directions
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
	Type       SortType
	Direction  SortDirection
	StyleIndex int
}

// Table style pairs to cycle through
var stylePairs = [][]table.Style{
	{table.StyleColoredBright, table.StyleColoredDark},
	{table.StyleColoredBlackOnBlueWhite, table.StyleColoredBlueWhiteOnBlack},
	{table.StyleColoredBlackOnCyanWhite, table.StyleColoredCyanWhiteOnBlack},
	{table.StyleColoredBlackOnGreenWhite, table.StyleColoredGreenWhiteOnBlack},
	{table.StyleColoredBlackOnMagentaWhite, table.StyleColoredMagentaWhiteOnBlack},
	{table.StyleColoredBlackOnRedWhite, table.StyleColoredRedWhiteOnBlack},
	{table.StyleColoredBlackOnYellowWhite, table.StyleColoredYellowWhiteOnBlack},
}

func main() {
	// Determine default branch
	defaultBranch := "main"
	if !branchExists(defaultBranch) {
		defaultBranch = "master"
	}

	// Initialize with alphabetical sorting
	sortState := SortState{
		Type:       SortAlphabetical,
		Direction:  SortAscending,
		StyleIndex: 0,
	}

	displayBranchTable(defaultBranch, sortState)
}

var (
	colTitleIndex      = "#"
	colTitleBranch     = "Branch"
	colTitleLastCommit = "Last Commit"
	colTitleProgress   = "Ahead\\Behind"
	colTotleDelete     = "Delete"
	rowHeader          = table.Row{
		colTitleIndex,
		colTitleBranch,
		colTitleLastCommit,
		colTitleProgress,
		colTotleDelete,
	}
)

func displayBranchTable(defaultBranch string, sortState SortState) {
	// Import required for terminal raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("Error setting terminal to raw mode: %v\n", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	for {
		clearScreen()
		renderTable(defaultBranch, sortState)

		fmt.Print("┌──────────────────────────────────────────────────┐\r\n")
		fmt.Print("│ [a] Alphabetical  [c] Creation  [u] Updated      │\r\n")
		fmt.Print("│ [r] Reverse sort  [s] Style     [q] Quit         │\r\n")
		fmt.Print("└──────────────────────────────────────────────────┘\r\n")

		// Read a single byte without waiting for Enter
		buf := make([]byte, 1)
		_, err := os.Stdin.Read(buf)
		if err != nil {
			fmt.Printf("Error reading from stdin: %v\r\n", err)
			return
		}

		input := buf[0]

		// Process input immediately
		switch input {
		case 'a':
			sortState.Type = SortAlphabetical
		case 'c':
			sortState.Type = SortCreationDate
		case 'u':
			sortState.Type = SortCommitDate
		case 'r':
			if sortState.Direction == SortAscending {
				sortState.Direction = SortDescending
			} else {
				sortState.Direction = SortAscending
			}
		case 's':
			sortState.StyleIndex = (sortState.StyleIndex + 1) % len(stylePairs)
		case 'q', 3: // 'q' or Ctrl+C to quit
			return
		}
	}
}

func renderTable(defaultBranch string, sortState SortState) {
	t := table.NewWriter()
	// Instead of directly mirroring to stdout, we'll capture the output
	var tableOutput strings.Builder
	t.SetOutputMirror(&tableOutput)

	// Set the style based on the current style index
	styleIdx := sortState.StyleIndex % len(stylePairs)
	currentStyle := stylePairs[styleIdx][0]
	t.SetStyle(currentStyle)
	colorBOnW := text.Colors{text.BgWhite, text.FgBlack}

	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: colTitleIndex, Colors: text.Colors{text.FgWhite}, ColorsHeader: colorBOnW},
		{Name: colTitleBranch, Colors: text.Colors{text.FgWhite}, ColorsHeader: colorBOnW},
		{Name: colTitleLastCommit, Colors: text.Colors{text.FgWhite}, ColorsHeader: colorBOnW, ColorsFooter: colorBOnW},
		{Name: colTitleProgress, Colors: text.Colors{text.FgGreen}, ColorsHeader: colorBOnW, ColorsFooter: colorBOnW},
		{Name: colTotleDelete, Align: text.AlignRight, Colors: text.Colors{text.FgRed}, ColorsHeader: colorBOnW, ColorsFooter: colorBOnW},
	})

	t.AppendHeader(rowHeader)

	// Get branches and sort them
	branches := getLocalBranches()
	branches = sortBranches(branches, sortState)

	for i, br := range branches {
		commitTime := getLastCommitTime(br)
		timeStr := commitTime.Format("2006-01-02 15:04")

		ahead, behind := getAheadBehind(defaultBranch, br)
		track := fmt.Sprintf("%d/%d", ahead, behind)

		// 'delete' hyperlink triggers this app with args
		link := fmt.Sprintf("\x1b]8;;git-branches delete %s\x1b\\delete\x1b]8;;\x1b\\", br)

		t.AppendRow(table.Row{i + 1, br, timeStr, track, link})
	}

	t.Render()

	// Now print the table with explicit carriage returns and line feeds
	lines := strings.Split(tableOutput.String(), "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Printf("%s\r\n", line)
		}
	}
}

// Sort branches based on the current sort state
func sortBranches(branches []string, sortState SortState) []string {
	switch sortState.Type {
	case SortAlphabetical:
		if sortState.Direction == SortAscending {
			sort.Strings(branches)
		} else {
			sort.Sort(sort.Reverse(sort.StringSlice(branches)))
		}

	case SortCreationDate:
		sort.SliceStable(branches, func(i, j int) bool {
			timeI := getBranchCreationTime(branches[i])
			timeJ := getBranchCreationTime(branches[j])

			if sortState.Direction == SortAscending {
				return timeI.Before(timeJ)
			}
			return timeI.After(timeJ)
		})

	case SortCommitDate:
		sort.SliceStable(branches, func(i, j int) bool {
			timeI := getLastCommitTime(branches[i])
			timeJ := getLastCommitTime(branches[j])

			if sortState.Direction == SortAscending {
				return timeI.Before(timeJ)
			}
			return timeI.After(timeJ)
		})
	}

	return branches
}

// Get branch creation time
func getBranchCreationTime(branch string) time.Time {
	cmd := exec.Command("git", "log", "--format=%ct", "--reverse", branch, "-n", "1")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}
	sec, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return time.Unix(sec, 0)
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// Rest of the existing functions...
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
