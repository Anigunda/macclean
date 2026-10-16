// Package tui is the interactive terminal interface for macclean,
// built on Charm (Bubble Tea + Bubbles + Lipgloss).
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Anigunda/macclean/internal/cleaner"
)

type stage int

const (
	stageScanning stage = iota
	stageReady
	stageConfirm
	stageCleaning
	stageDone
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sizeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
)

type scanDoneMsg struct{ items []cleaner.ItemEntry }

type cleanDoneMsg struct {
	freed   int64
	removed int64
	failed  int64
	mode    string // "clean" or "quarantine"
	batchID string
}

type model struct {
	stage      stage
	spinner    spinner.Model
	table      table.Model
	items      []cleaner.ItemEntry
	selected   map[int]bool
	targets    []cleaner.Target
	width      int
	height     int
	freed      int64
	failed     int64
	quarantine bool // pending action is quarantine instead of delete
	doneMode   string
	batchID    string
}

// Run starts the interactive program over the given targets.
// It should only be called on a TTY.
func Run(targets []cleaner.Target) error {
	_, err := tea.NewProgram(initialModel(targets)).Run()
	return err
}

func initialModel(targets []cleaner.Target) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		stage:    stageScanning,
		spinner:  sp,
		selected: map[int]bool{},
		targets:  targets,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, scanCmd(m.targets))
}

func scanCmd(targets []cleaner.Target) tea.Cmd {
	return func() tea.Msg {
		return scanDoneMsg{items: cleaner.CollectItems(targets)}
	}
}

func quarantineCmd(entries []cleaner.ItemEntry) tea.Cmd {
	return func() tea.Msg {
		out := cleaner.QuarantineEntries(entries)
		if out.Moved > 0 {
			_ = cleaner.AppendHistory(cleaner.Event{
				Time: time.Now(), Mode: "tui-quarantine",
				Freed: out.Bytes, Removed: out.Moved, Failed: int64(out.Failed), Paths: out.Paths,
			})
		}
		return cleanDoneMsg{freed: out.Bytes, removed: out.Moved, failed: int64(out.Failed),
			mode: "quarantine", batchID: out.BatchID}
	}
}

func cleanCmd(entries []cleaner.ItemEntry) tea.Cmd {
	return func() tea.Msg {
		res := cleaner.CleanEntries(entries)
		if res.Removed > 0 || res.Failed > 0 {
			_ = cleaner.AppendHistory(cleaner.Event{
				Time: time.Now(), Mode: "tui",
				Freed: res.Freed, Removed: res.Removed, Failed: res.Failed, Paths: res.Paths,
			})
		}
		return cleanDoneMsg{freed: res.Freed, removed: res.Removed, failed: res.Failed, mode: "clean"}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.stage == stageReady || m.stage == stageConfirm {
			m.refreshTableSize()
		}
		return m, nil
	case scanDoneMsg:
		m.items = msg.items
		m.buildTable()
		m.stage = stageReady
		return m, nil
	case cleanDoneMsg:
		m.freed, m.failed = msg.freed, msg.failed
		m.doneMode, m.batchID = msg.mode, msg.batchID
		m.stage = stageDone
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.stage {
		case stageScanning:
			if msg.String() == "q" {
				return m, tea.Quit
			}
		case stageReady:
			return m.updateReady(msg)
		case stageConfirm:
			switch msg.String() {
			case "y", "Y":
				m.stage = stageCleaning
				if m.quarantine {
					return m, tea.Batch(m.spinner.Tick, quarantineCmd(m.selectedEntries()))
				}
				return m, tea.Batch(m.spinner.Tick, cleanCmd(m.selectedEntries()))
			default: // any other key cancels
				m.stage = stageReady
				return m, nil
			}
		case stageDone:
			if msg.String() == "q" || msg.String() == "enter" {
				return m, tea.Quit
			}
		}
	}

	// Forward remaining messages to the active component.
	switch m.stage {
	case stageScanning, stageCleaning:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case stageReady:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateReady(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case " ":
		i := m.table.Cursor()
		if i >= 0 && i < len(m.items) {
			if m.selected[i] {
				delete(m.selected, i)
			} else {
				m.selected[i] = true
			}
			m.table.SetRows(m.rows())
		}
		return m, nil
	case "a":
		if len(m.selected) == len(m.items) {
			m.selected = map[int]bool{}
		} else {
			for i := range m.items {
				m.selected[i] = true
			}
		}
		m.table.SetRows(m.rows())
		return m, nil
	case "c", "enter":
		if len(m.selected) == 0 {
			return m, nil
		}
		m.quarantine = false
		m.stage = stageConfirm
		return m, nil
	case "m":
		if len(m.selected) == 0 {
			return m, nil
		}
		m.quarantine = true
		m.stage = stageConfirm
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	switch m.stage {
	case stageScanning:
		return fmt.Sprintf("\n  %s Scanning your Mac...\n\n", m.spinner.View())
	case stageReady:
		return m.readyView()
	case stageConfirm:
		return m.confirmView()
	case stageCleaning:
		return fmt.Sprintf("\n  %s Cleaning...\n\n", m.spinner.View())
	case stageDone:
		return m.doneView()
	}
	return ""
}

func (m model) readyView() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s\n\n", titleStyle.Render("macclean — select what to delete"))
	b.WriteString(m.table.View())

	count, bytes := m.selectedSummary()
	fmt.Fprintf(&b, "\n\n  Selected %d items (%s)\n", count, sizeStyle.Render(cleaner.HumanSize(bytes)))
	if cur := m.table.Cursor(); cur >= 0 && cur < len(m.items) {
		it := m.items[cur]
		fmt.Fprintf(&b, "  %s\n", helpStyle.Render(
			cleaner.ShortPath(it.Path)+"  —  "+it.Reason))
	}
	fmt.Fprintf(&b, "  %s\n",
		helpStyle.Render("↑/↓ move • space toggle • a all • c clean • m quarantine • q quit"))
	return b.String()
}

func (m model) confirmView() string {
	count, bytes := m.selectedSummary()
	var question string
	if m.quarantine {
		question = fmt.Sprintf("Quarantine %d items (%s)?\n\n  They move to ~/.macclean/quarantine and can be restored\n  with 'macclean undo'. [y/N]", count, cleaner.HumanSize(bytes))
	} else {
		question = fmt.Sprintf("Delete %d items permanently, freeing %s? [y/N]", count, cleaner.HumanSize(bytes))
	}
	return fmt.Sprintf("\n  %s\n\n  %s\n\n  %s\n",
		titleStyle.Render("macclean"),
		warnStyle.Render(question),
		helpStyle.Render("y confirm • any other key to go back"))
}

func (m model) doneView() string {
	var msg, hint string
	if m.doneMode == "quarantine" {
		msg = fmt.Sprintf("Quarantined %s", cleaner.HumanSize(m.freed))
		hint = fmt.Sprintf("batch %s - restore with 'macclean undo' (auto-purges after %d days)",
			m.batchID, cleaner.GetConfig().QuarantineDaysOrDefault())
	} else {
		msg = fmt.Sprintf("Freed %s", cleaner.HumanSize(m.freed))
		hint = "q or enter to quit"
	}
	if m.failed > 0 {
		msg += fmt.Sprintf(" (%d locked/in-use items skipped)", m.failed)
	}
	return fmt.Sprintf("\n  %s\n\n  %s\n\n  %s\n",
		titleStyle.Render("Done"),
		okStyle.Render(msg),
		helpStyle.Render(hint))
}

func (m model) selectedSummary() (count int, bytes int64) {
	for idx := range m.selected {
		if idx < len(m.items) {
			count++
			bytes += m.items[idx].Size
		}
	}
	return count, bytes
}

func (m model) selectedEntries() []cleaner.ItemEntry {
	out := make([]cleaner.ItemEntry, 0, len(m.selected))
	for idx := range m.selected {
		if idx < len(m.items) {
			out = append(out, m.items[idx])
		}
	}
	return out
}

// --- table ---

func (m *model) buildTable() {
	// Pre-select everything suggested to delete.
	m.selected = map[int]bool{}
	for i, it := range m.items {
		if !it.Keep {
			m.selected[i] = true
		}
	}
	t := table.New(
		table.WithColumns(m.columns()),
		table.WithRows(m.rows()),
		table.WithFocused(true),
		table.WithHeight(m.tableHeight()),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	m.table = t
}

func (m model) columns() []table.Column {
	w := m.width
	if w <= 0 {
		w = 100
	}
	selW, sizeW, actW, catW := 4, 10, 7, 14
	rest := w - selW - sizeW - actW - catW - 14 // cell padding + borders
	pathW := rest * 3 / 5
	reasonW := rest - pathW
	if pathW < 16 {
		pathW = 16
	}
	if reasonW < 8 {
		reasonW = 8
	}
	return []table.Column{
		{Title: "SEL", Width: selW},
		{Title: "SIZE", Width: sizeW},
		{Title: "ACTION", Width: actW},
		{Title: "CATEGORY", Width: catW},
		{Title: "PATH", Width: pathW},
		{Title: "WHY", Width: reasonW},
	}
}

func (m model) colWidth(title string, fallback int) int {
	for _, c := range m.columns() {
		if c.Title == title {
			return c.Width
		}
	}
	return fallback
}

func (m model) tableHeight() int {
	h := m.height - 9 // title, footer, padding
	if h < 5 {
		h = 5
	}
	return h
}

func (m model) rows() []table.Row {
	rows := make([]table.Row, len(m.items))
	pathW := m.colWidth("PATH", 30)
	reasonW := m.colWidth("WHY", 20)
	for i, it := range m.items {
		check := "[ ]"
		if m.selected[i] {
			check = "[x]"
		}
		action := "delete" // plain text: the table component is not ANSI-safe
		if it.Keep {
			action = "keep"
		}
		rows[i] = table.Row{
			check,
			cleaner.HumanSize(it.Size),
			action,
			truncate(it.TargetName, 14),
			truncate(cleaner.ShortPath(it.Path), pathW),
			truncate(it.Reason, reasonW),
		}
	}
	return rows
}

func (m *model) refreshTableSize() {
	m.table.SetColumns(m.columns())
	m.table.SetRows(m.rows())
	m.table.SetHeight(m.tableHeight())
	m.table.SetWidth(m.width)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if len(r) > w-1 {
		r = r[:w-1]
	}
	return string(r) + "…"
}
