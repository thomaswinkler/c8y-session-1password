package picker

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

// isInCommandSubstitution detects if we're running in command substitution context
// GetTerminalColorProfile determines the best color profile for the terminal (exported version)
func GetTerminalColorProfile() termenv.Profile {
	return getTerminalColorProfile()
}

// getTerminalColorProfile determines the best color profile for the terminal
func getTerminalColorProfile() termenv.Profile {
	// First, get the natural color profile detection
	profile := termenv.ColorProfile()

	// Debug output
	if os.Getenv("C8YOP_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "getTerminalColorProfile: initial profile=%v, inCommandSub=%v, TERM=%s\n",
			profile, isInCommandSubstitution(), os.Getenv("TERM"))
	}

	// If we're in command substitution and colors are disabled,
	// we need to determine what the terminal would support if it were direct
	if profile == termenv.Ascii && isInCommandSubstitution() {
		// Check terminal type to determine appropriate color support
		termType := os.Getenv("TERM")

		var newProfile termenv.Profile
		switch {
		case strings.Contains(termType, "256color"):
			newProfile = termenv.ANSI256
		case strings.Contains(termType, "color"):
			newProfile = termenv.ANSI
		case termType == "xterm-256color" || termType == "screen-256color":
			newProfile = termenv.ANSI256
		case termType == "xterm" || termType == "screen":
			newProfile = termenv.ANSI
		default:
			// Conservative fallback for unknown terminals
			newProfile = termenv.ANSI
		}

		if os.Getenv("C8YOP_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "getTerminalColorProfile: overriding %v -> %v for term=%s\n",
				profile, newProfile, termType)
		}

		return newProfile
	}

	return profile
}

func isInCommandSubstitution() bool {
	// Check if stdout is not a terminal (common in command substitution)
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// If output is being piped or redirected, we're likely in command substitution
	mode := stat.Mode()
	if (mode & os.ModeCharDevice) == 0 {
		return true
	}

	// Additional check: look for common command substitution patterns
	// Check if SHLVL indicates we're in a subshell
	if shlvl := os.Getenv("SHLVL"); shlvl != "" && shlvl != "1" {
		// This is a heuristic - we might be in a subshell
		return true
	}

	return false
}

// PickerMetadata holds information about the query parameters used
type PickerMetadata struct {
	Vaults  []string
	Tags    []string
	Filter  string
	NoColor bool
}

var (
	appStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#119D11", // Green text for light terminals
			Dark:  "#FFBE00", // Yellow text for dark terminals
		})

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{
			Light: "#056AD6", // Blue for light terminals
			Dark:  "#56C8FF", // Lighter blue for dark terminals
		}).
		Render
)

type listKeyMap struct {
	toggleSpinner    key.Binding
	toggleTitleBar   key.Binding
	toggleStatusBar  key.Binding
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	insertItem       key.Binding
	selectItem       key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		insertItem: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add item"),
		),
		toggleSpinner: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "toggle spinner"),
		),
		toggleTitleBar: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle title"),
		),
		toggleStatusBar: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle status"),
		),
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
		selectItem: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
	}
}

type model struct {
	list          list.Model
	itemGenerator *randomItemGenerator
	keys          *listKeyMap
	delegateKeys  *delegateKeyMap
	wasSelected   bool
	metadata      PickerMetadata
}

func newModel(itemGenerator randomItemGenerator, metadata PickerMetadata) model {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	// Make initial list of items
	items := make([]list.Item, itemGenerator.Len())
	for i := 0; i < itemGenerator.Len(); i++ {
		items[i] = itemGenerator.Next()
	}

	// Setup list
	delegate := newItemDelegate(delegateKeys)
	sessionList := list.New(items, delegate, 0, 0)

	// Build title with metadata information
	title := buildTitle(itemGenerator.Len(), metadata)
	sessionList.Title = title
	sessionList.Styles.Title = titleStyle

	// Hide the status bar by default (which shows "X items")
	sessionList.SetShowStatusBar(false)

	sessionList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleSpinner,
			listKeys.insertItem,
			listKeys.toggleTitleBar,
			listKeys.toggleStatusBar,
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
			listKeys.selectItem,
		}
	}

	return model{
		list:          sessionList,
		keys:          listKeys,
		delegateKeys:  delegateKeys,
		itemGenerator: &itemGenerator,
		metadata:      metadata,
	}
}

func (m model) WasSelected() bool {
	return m.wasSelected
}

func (m model) Init() tea.Cmd {
	var profile termenv.Profile

	// Check if colors should be disabled
	if m.metadata.NoColor {
		profile = termenv.Ascii
	} else {
		// Use our improved color detection
		profile = getTerminalColorProfile()
	}

	lipgloss.SetColorProfile(profile)

	// Log color profile for debugging if debug mode is enabled
	if os.Getenv("C8YOP_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "Terminal: %s, Color profile: %v, NoColor: %v, InCommandSubstitution: %v\n",
			os.Getenv("TERM"), profile, m.metadata.NoColor, isInCommandSubstitution())
	}

	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleSpinner):
			cmd := m.list.ToggleSpinner()
			return m, cmd

		case key.Matches(msg, m.keys.toggleTitleBar):
			v := !m.list.ShowTitle()
			m.list.SetShowTitle(v)
			m.list.SetShowFilter(v)
			m.list.SetFilteringEnabled(v)
			return m, nil

		case key.Matches(msg, m.keys.toggleStatusBar):
			m.list.SetShowStatusBar(!m.list.ShowStatusBar())
			return m, nil

		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil

		case key.Matches(msg, m.keys.insertItem):
			m.delegateKeys.remove.SetEnabled(true)
			newItem := m.itemGenerator.Next()
			insCmd := m.list.InsertItem(0, newItem)
			statusCmd := m.list.NewStatusMessage(statusMessageStyle("Added " + newItem.Title()))
			return m, tea.Batch(insCmd, statusCmd)

		case key.Matches(msg, m.keys.selectItem):
			m.wasSelected = true
			return m, tea.Quit
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return appStyle.Render(m.list.View())
}

func Pick(sessions []*core.CumulocitySession, metadata PickerMetadata) (*core.CumulocitySession, error) {
	itemGenerator := randomItemGenerator{
		sessions: sessions,
	}

	m, err := tea.NewProgram(newModel(itemGenerator, metadata), tea.WithAltScreen(), tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		os.Exit(1)
	}

	session := m.(model)

	if session.WasSelected() {
		if selectedSession, ok := session.list.SelectedItem().(*core.CumulocitySession); ok {
			return selectedSession, nil
		}
	}

	return nil, fmt.Errorf("empty")
}

func (pm PickerMetadata) String() string {
	var b strings.Builder

	if len(pm.Vaults) > 0 {
		b.WriteString("Vaults: " + strings.Join(pm.Vaults, ", ") + "\n")
	}

	if len(pm.Tags) > 0 {
		b.WriteString("Tags: " + strings.Join(pm.Tags, ", ") + "\n")
	}

	if pm.Filter != "" {
		b.WriteString("Filter: " + pm.Filter + "\n")
	}

	return b.String()
}

// buildTitle creates a descriptive title showing session count, vaults, and tags
func buildTitle(sessionCount int, metadata PickerMetadata) string {
	parts := []string{fmt.Sprintf("Sessions (%d)", sessionCount)}

	if len(metadata.Vaults) > 0 {
		if len(metadata.Vaults) == 1 {
			parts = append(parts, fmt.Sprintf("Vault: %s", metadata.Vaults[0]))
		} else {
			parts = append(parts, fmt.Sprintf("Vaults: %s", strings.Join(metadata.Vaults, ", ")))
		}
	} else {
		parts = append(parts, "All Vaults")
	}

	if len(metadata.Tags) > 0 {
		if len(metadata.Tags) == 1 {
			parts = append(parts, fmt.Sprintf("Tag: %s", metadata.Tags[0]))
		} else {
			parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(metadata.Tags, ", ")))
		}
	}

	if metadata.Filter != "" {
		parts = append(parts, fmt.Sprintf("Filter: %s", metadata.Filter))
	}

	return strings.Join(parts, " • ")
}
