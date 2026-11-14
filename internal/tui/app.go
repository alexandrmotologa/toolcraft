package tui

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/domain"
	"github.com/alexandrmotologa/toolcraft/internal/generator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the Bubbletea application state.
type Model struct {
	Spec       *domain.Spec
	Report     *domain.HardeningReport
	Config     *config.Config
	Cursor     int
	ActivePane int // 0: Tools list, 1: Details pane
	Width      int
	Height     int
	StatusMsg  string
}

// NewModel initializes the TUI model.
func NewModel(spec *domain.Spec, report *domain.HardeningReport, cfg *config.Config) Model {
	return Model{
		Spec:       spec,
		Report:     report,
		Config:     cfg,
		Cursor:     0,
		ActivePane: 0,
		Width:      100,
		Height:     30,
		StatusMsg:  "Ready. [Space]: toggle tool, [g]: generate Go, [t]: generate TS, [q]: quit",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			if m.Cursor < len(m.Spec.Tools)-1 {
				m.Cursor++
			}

		case "tab":
			m.ActivePane = (m.ActivePane + 1) % 2

		case " ":
			if len(m.Spec.Tools) > 0 {
				m.Spec.Tools[m.Cursor].Enabled = !m.Spec.Tools[m.Cursor].Enabled
				m.StatusMsg = fmt.Sprintf("Toggled tool '%s' -> enabled=%v", m.Spec.Tools[m.Cursor].ID, m.Spec.Tools[m.Cursor].Enabled)
			}

		case "a":
			allEnabled := true
			for _, t := range m.Spec.Tools {
				if !t.Enabled {
					allEnabled = false
					break
				}
			}
			for i := range m.Spec.Tools {
				m.Spec.Tools[i].Enabled = !allEnabled
			}
			m.StatusMsg = fmt.Sprintf("All tools toggled: %v", !allEnabled)

		case "g":
			// Generate Go
			gen, err := generator.GetGenerator("go")
			if err != nil {
				m.StatusMsg = "Error: " + err.Error()
				return m, nil
			}
			m.Config.Output.Language = "go"
			activeSpec := m.filterActiveTools()
			if err := gen.Generate(activeSpec, m.Config); err != nil {
				m.StatusMsg = "Generation failed: " + err.Error()
			} else {
				m.StatusMsg = fmt.Sprintf("Successfully generated Go MCP server in %s", m.Config.Output.OutputDir)
			}

		case "t":
			// Generate TypeScript
			gen, err := generator.GetGenerator("typescript")
			if err != nil {
				m.StatusMsg = "Error: " + err.Error()
				return m, nil
			}
			m.Config.Output.Language = "typescript"
			activeSpec := m.filterActiveTools()
			if err := gen.Generate(activeSpec, m.Config); err != nil {
				m.StatusMsg = "Generation failed: " + err.Error()
			} else {
				m.StatusMsg = fmt.Sprintf("Successfully generated TypeScript MCP package in %s", m.Config.Output.OutputDir)
			}
		}
	}

	return m, nil
}

func (m Model) filterActiveTools() *domain.Spec {
	filtered := *m.Spec
	filtered.Tools = make([]domain.ToolDefinition, 0)
	for _, t := range m.Spec.Tools {
		if t.Enabled {
			filtered.Tools = append(filtered.Tools, t)
		}
	}
	return &filtered
}

func (m Model) View() string {
	header := HeaderStyle.Render(fmt.Sprintf(" toolcraft TUI | %s v%s | Operations: %d | Token Savings: %.1f%% ",
		m.Spec.Title, m.Spec.Version, len(m.Spec.Tools), m.Report.ReductionPercent))

	leftWidth := m.Width * 4 / 10
	if leftWidth < 35 {
		leftWidth = 35
	}
	rightWidth := m.Width - leftWidth - 6
	if rightWidth < 40 {
		rightWidth = 40
	}

	paneHeight := m.Height - 6
	if paneHeight < 10 {
		paneHeight = 10
	}

	leftView := m.renderToolList(leftWidth, paneHeight)
	rightView := m.renderToolDetails(rightWidth, paneHeight)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftView, " ", rightView)
	footer := FooterStyle.Render(m.StatusMsg)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) renderToolList(width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Available MCP Tools:") + "\n\n")

	for i, t := range m.Spec.Tools {
		cursor := "  "
		if i == m.Cursor {
			cursor = "> "
		}

		check := "[x]"
		if !t.Enabled {
			check = "[ ]"
		}

		methodBadge := formatMethod(t.Method)
		line := fmt.Sprintf("%s%s %s %s", cursor, check, methodBadge, t.ID)

		if i == m.Cursor {
			b.WriteString(SelectedItemStyle.Width(width - 4).Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	style := ListPaneStyle.Width(width).Height(height)
	if m.ActivePane == 0 {
		style = ActivePaneStyle.Width(width).Height(height)
	}
	return style.Render(b.String())
}

func (m Model) renderToolDetails(width, height int) string {
	if len(m.Spec.Tools) == 0 || m.Cursor >= len(m.Spec.Tools) {
		return DetailPaneStyle.Width(width).Height(height).Render("No tool selected")
	}

	t := m.Spec.Tools[m.Cursor]

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Tool: "+t.ID) + "\n")
	b.WriteString(fmt.Sprintf("Endpoint: %s %s\n", formatMethod(t.Method), t.Path))
	b.WriteString(fmt.Sprintf("Read-Only: %v | Original ID: %s\n\n", t.ReadOnly, t.OriginalID))

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Model Description:") + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(ColorFg).Render(t.Description) + "\n\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Hardened Parameters:") + "\n")
	if len(t.Parameters) == 0 && (t.RequestBody == nil || len(t.RequestBody.Properties) == 0) {
		b.WriteString("  (None)\n")
	} else {
		for _, p := range t.Parameters {
			reqStr := "optional"
			if p.Required {
				reqStr = BadgeWarning.Render("required")
			}
			b.WriteString(fmt.Sprintf("  * %s (%s, in:%s) - %s\n", p.Name, p.Type, p.In, reqStr))
			if p.Description != "" {
				b.WriteString(fmt.Sprintf("    %s\n", p.Description))
			}
		}
		if t.RequestBody != nil {
			for _, p := range t.RequestBody.Properties {
				reqStr := "optional"
				if p.Required {
					reqStr = BadgeWarning.Render("required")
				}
				b.WriteString(fmt.Sprintf("  * [body] %s (%s) - %s\n", p.Name, p.Type, reqStr))
			}
		}
	}

	style := DetailPaneStyle.Width(width).Height(height)
	if m.ActivePane == 1 {
		style = ActivePaneStyle.Width(width).Height(height)
	}
	return style.Render(b.String())
}

func formatMethod(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return MethodGetStyle.Render("GET")
	case "POST":
		return MethodPostStyle.Render("POST")
	case "PUT":
		return MethodPutStyle.Render("PUT")
	case "DELETE":
		return MethodDeleteStyle.Render("DEL")
	default:
		return method
	}
}

// RunApp starts the Bubbletea program loop.
func RunApp(spec *domain.Spec, report *domain.HardeningReport, cfg *config.Config) error {
	p := tea.NewProgram(NewModel(spec, report, cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
