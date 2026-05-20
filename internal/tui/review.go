package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	tuitheme "github.com/usewhale/whale/internal/tui/theme"
)

type reviewMenuItem struct {
	Name        string
	Description string
	Action      string
	Prefill     string
}

func reviewMenuItems() []reviewMenuItem {
	return []reviewMenuItem{
		{Name: "Local changes", Description: "Review staged, unstaged, and relevant untracked files.", Action: "/review local"},
		{Name: "Current branch vs default branch", Description: "Review committed branch changes against the default branch.", Action: "/review branch"},
		{Name: "Pull request...", Description: "Review a GitHub PR by number or URL.", Prefill: "/review pr "},
		{Name: "Commit...", Description: "Review one commit by SHA.", Prefill: "/review commit "},
		{Name: "Custom instructions...", Description: "Describe exactly what to review.", Prefill: "/review "},
	}
}

func (m *model) handleReviewMenuKey(msg tea.KeyMsg) tea.Cmd {
	items := reviewMenuItems()
	switch msg.String() {
	case "esc", "ctrl+c":
		m.closeReviewMenu()
	case "up", "k":
		if m.reviewMenu.selected > 0 {
			m.reviewMenu.selected--
		}
	case "down", "j":
		if m.reviewMenu.selected < len(items)-1 {
			m.reviewMenu.selected++
		}
	case "enter":
		if m.reviewMenu.selected < 0 || m.reviewMenu.selected >= len(items) {
			return nil
		}
		item := items[m.reviewMenu.selected]
		if item.Action != "" {
			m.closeReviewMenu()
			return m.submitPrompt(item.Action)
		}
		if item.Prefill != "" {
			m.closeReviewMenu()
			m.input.SetValue(item.Prefill)
			m.skillBinding = nil
			m.resetHistoryNavigation()
			m.updateSlashMatches()
			m.refreshViewportContent()
		}
	}
	return nil
}

func (m *model) closeReviewMenu() {
	m.mode = modeChat
	m.reviewMenu.selected = 0
	m.status = "ready"
}

func (m model) renderReviewMenu() string {
	title := lipgloss.NewStyle().Foreground(tuitheme.Default.InfoSoft).Bold(true)
	muted := lipgloss.NewStyle().Foreground(tuitheme.Default.Muted)
	rows := []string{
		title.Render("Review"),
		muted.Render("Choose what to review"),
		"",
	}
	for i, item := range reviewMenuItems() {
		rows = append(rows, renderReviewMenuRow(item, i == m.reviewMenu.selected))
	}
	rows = append(rows, "", muted.Render("  ↑/↓ select · Enter confirm · Esc close"))
	return strings.Join(rows, "\n")
}

func renderReviewMenuRow(item reviewMenuItem, selected bool) string {
	muted := lipgloss.NewStyle().Foreground(tuitheme.Default.Muted)
	nameStyle := lipgloss.NewStyle()
	prefix := muted.Render("  ")
	if selected {
		prefix = lipgloss.NewStyle().Foreground(tuitheme.Default.InfoSoft).Bold(true).Render("> ")
		nameStyle = nameStyle.Foreground(tuitheme.Default.InfoSoft).Bold(true)
	}
	head := prefix + nameStyle.Render(item.Name)
	desc := strings.TrimSpace(item.Description)
	if desc == "" {
		return head
	}
	const descCol = 38
	gap := descCol - lipgloss.Width(head)
	if gap < 1 {
		gap = 1
	}
	return head + strings.Repeat(" ", gap) + muted.Render(desc)
}
