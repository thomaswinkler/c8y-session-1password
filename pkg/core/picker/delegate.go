package picker

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	// Set custom selection highlight colors with adaptive support
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#056AD6")). // Blue text for both light and dark terminals
		Background(lipgloss.Color("")).        // No background
		BorderForeground(lipgloss.Color("#056AD6")).
		Bold(true)

	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.AdaptiveColor{
			Light: "#1F4E79", // Even darker blue for better readability in light terminals
			Dark:  "#3A8BDB", // Keep existing lighter blue for dark terminals
		}).
		Background(lipgloss.Color("")). // No background
		BorderForeground(lipgloss.AdaptiveColor{
			Light: "#1F4E79", // Match border to description text color
			Dark:  "#3A8BDB",
		})

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		if i, ok := m.SelectedItem().(*core.CumulocitySession); ok {
			title = i.Host
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.choose):
				return m.NewStatusMessage(statusMessageStyle("You chose " + title))

			case key.Matches(msg, keys.remove):
				index := m.Index()
				m.RemoveItem(index)
				if len(m.Items()) == 0 {
					keys.remove.SetEnabled(false)
				}
				return m.NewStatusMessage(statusMessageStyle("Deleted " + title))
			case key.Matches(msg, keys.cancel):
				return tea.Quit
			}

		}

		return nil
	}

	help := []key.Binding{keys.choose, keys.remove}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

type delegateKeyMap struct {
	choose key.Binding
	remove key.Binding
	cancel key.Binding
}

// Additional short help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.choose,
		d.remove,
		d.cancel,
	}
}

// Additional full help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.choose,
			d.remove,
			d.cancel,
		},
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "choose"),
		),
		remove: key.NewBinding(
			key.WithKeys("x", "backspace"),
			key.WithHelp("x", "delete"),
		),
		cancel: key.NewBinding(
			key.WithKeys("esc", "ctrl+c", "c"),
			key.WithHelp("esc/ctrl+c/c", "cancel"),
		),
	}
}
