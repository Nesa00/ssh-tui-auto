package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StyleList struct {
	TitleStyle        lipgloss.Style
	ItemStyle         lipgloss.Style
	SelectedItemStyle lipgloss.Style
	PaginationStyle   lipgloss.Style
	HelpStyle         lipgloss.Style
	QuitTextStyle     lipgloss.Style
}

var (
	styleList = StyleList{
		TitleStyle:        lipgloss.NewStyle().MarginLeft(2),
		ItemStyle:         lipgloss.NewStyle().PaddingLeft(4),
		SelectedItemStyle: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("35")),
		PaginationStyle:   list.DefaultStyles().PaginationStyle.PaddingLeft(4),
		HelpStyle:         list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1),
		QuitTextStyle:     lipgloss.NewStyle().Margin(1, 0, 2, 4),
	}
)


type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	// fn := itemStyle.Render
	fn := styleList.ItemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			// return selectedItemStyle.Render("> " + strings.Join(s, " "))
			return styleList.SelectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type ListMenu struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m ListMenu) Init() tea.Cmd {
	return nil
}

func (m ListMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ListMenu) View() string {
	if m.choice != "" {
		// return quitTextStyle.Render(fmt.Sprintf("%s Selected", m.choice))
		return styleList.QuitTextStyle.Render(fmt.Sprintf("%s Selected", m.choice))
		// return quitTextStyle.Render("")
	}
	if m.quitting {
		return styleList.QuitTextStyle.Render("Exiting...")
		// return quitTextStyle.Render("Exiting...")
	}
	return "\n" + m.list.View()
}

func ListMenuTUI(itemsList []string) string {
	var items []list.Item
	for _, i := range itemsList {
		items = append(items, item(i))
	}

	const defaultWidth = 20
	const listHeight = 14

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Please select operation"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = styleList.TitleStyle
	l.Styles.PaginationStyle = styleList.PaginationStyle
	l.Styles.HelpStyle = styleList.HelpStyle

	m := ListMenu{list: l}

	p := tea.NewProgram(m)

	finalModel, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	m, ok := finalModel.(ListMenu)
	if !ok {
		fmt.Println("Error asserting final ListMenu")
		os.Exit(1)
	}

	return m.choice
}

// ################################################################################################

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type TypingInput struct {
	focusIndex int
	inputs     []textinput.Model
	cursorMode cursor.Mode
}

func initialModel(menu []string) TypingInput {
	m := TypingInput{
		inputs: make([]textinput.Model, len(menu)),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = menu[i]
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case i:
			t.Placeholder = menu[i]
			t.CharLimit = 64
		}
		m.inputs[i] = t
	}

	return m
}

func (m TypingInput) Init() tea.Cmd {
	return textinput.Blink
}

func (m TypingInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Change cursor mode
		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				cmds[i] = m.inputs[i].Cursor.SetMode(m.cursorMode)
			}
			return m, tea.Batch(cmds...)

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()
			if s == "enter" && m.focusIndex == len(m.inputs) {
				// // add the input to the output
				// for i := range m.inputs {
				// 	m.output = append(m.output, m.inputs[i].Value())
				// }
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *TypingInput) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m TypingInput) View() string {
	var b strings.Builder

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render("cursor mode is "))
	b.WriteString(cursorModeHelpStyle.Render(m.cursorMode.String()))
	b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))

	return b.String()
}

func TextInputTUI(text []string) map[string]string {
	p := tea.NewProgram(initialModel(text))
	finalModel, err := p.StartReturningModel()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	m, ok := finalModel.(TypingInput)
	if !ok {
		fmt.Println("Error asserting final TypingInput")
		os.Exit(1)
	}

	res := make(map[string]string)
	for i := range m.inputs {
		res[text[i]] = m.inputs[i].Value()
	}

	return res
}
