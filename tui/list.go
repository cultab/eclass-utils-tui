package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/Huray-hub/eclass-utils/assignments/assignment"
	"github.com/Huray-hub/eclass-utils/assignments/cmd/flags"
	"github.com/Huray-hub/eclass-utils/assignments/config"
	"github.com/Huray-hub/eclass-utils/assignments/course"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listModel struct {
	list              list.Model
	cache             []item
	showHidden        bool
	hiddenAssignments map[string]struct{}
	hiddenCourses     map[string]struct{}
	keys              keyBinds
}

func NewList() listModel {
	m := listModel{
		list:              list.New([]list.Item{}, itemDelegate{}, 0, 0),
		showHidden:        false,
		hiddenAssignments: make(map[string]struct{}),
		hiddenCourses:     make(map[string]struct{}),
		keys:              newKeyBinds(),
	}
	m.list.Title = "Εργασίες"
	m.list.SetShowStatusBar(true)
    statusTime, err := time.ParseDuration("5s")
    if err != nil {
        log.Fatal("Parsing status message duration failed.")
    }
    m.list.StatusMessageLifetime = statusTime
	m.list.SetStatusBarItemName("Εργασία", "Εργασίες")
	m.list.SetSpinner(spinner.Dot)
	m.list.Styles.Spinner = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	m.list.StartSpinner()
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{

			m.keys.toggleHidden,
			m.keys.toggleHideCourse,
			m.keys.toggleHideAssignment,
		}
	}
	return m
}

type keyBinds struct {
	toggleHideAssignment key.Binding
	toggleHideCourse     key.Binding
	toggleHidden         key.Binding
}

func newKeyBinds() keyBinds {
	return keyBinds{
		toggleHideAssignment: key.NewBinding(
			key.WithKeys("c", "ψ"),
			key.WithHelp("c|ψ", "Κρύψε εργασία"),
		),
		toggleHideCourse: key.NewBinding(
			key.WithKeys("x", "θ"),
			key.WithHelp("x|θ", "Κρύψε μάθημα"),
		),
		toggleHidden: key.NewBinding(
			key.WithKeys(tea.KeySpace.String()),
			key.WithHelp("space", "Εμφάνησε κρυμένες εργασίες"),
		),
	}
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(
		m.list.StartSpinner(),
		getAssignments,
		mockGetAssignments,
	)
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type updateMsg struct{}

func updateCmd() tea.Msg { return updateMsg{} }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.toggleHideAssignment):
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				log.Print("Type Assertion failed")
			}
			// toggle hidden
			hidden := false
			for hidden_ass := range m.hiddenAssignments {
				if hidden_ass == i.assignment.ID {
					hidden = true
				}
			}
			if hidden {
				delete(m.hiddenAssignments, i.assignment.ID)
			} else {
				m.hiddenAssignments[i.assignment.ID] = struct{}{}
			}
			statusCmd := m.list.NewStatusMessage("Έκρυψες την εργασία " + i.assignment.Title + ".")
			return m, tea.Batch(updateCmd, statusCmd)
		case key.Matches(msg, m.keys.toggleHideCourse):
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				log.Print("Type Assertion failed")
			}
			// toggle hidden
			hidden := false
			for hidden_ass := range m.hiddenCourses {
				if hidden_ass == i.assignment.Course.ID {
					hidden = true
				}
			}
			if hidden {
				delete(m.hiddenCourses, i.assignment.Course.ID)
			} else {
				m.hiddenCourses[i.assignment.Course.ID] = struct{}{}
			}
			statusCmd := m.list.NewStatusMessage("Έκρυψες τις εργασίες του μαθήματος " + i.assignment.Course.Name + ".")
			return m, tea.Batch(updateCmd, statusCmd)
		case key.Matches(msg, m.keys.toggleHidden):
			m.showHidden = !m.showHidden
			return m, updateCmd
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case updateMsg:
		log.Print("Update list of assignments from cache")
		for item := range m.list.Items() { // remove all items
			m.list.RemoveItem(item)
		}
		var new_items []list.Item
		for _, item := range m.cache {
			var hidden = false
			for hidden_ass := range m.hiddenAssignments {
				if hidden_ass == item.assignment.ID {
					hidden = true
				}
			}
			for hidden_course := range m.hiddenCourses {
				if hidden_course == item.assignment.Course.ID {
					hidden = true
				}
			}

			if hidden == m.showHidden {
				new_items = append(new_items, item)
			}

		}
		m.list.SetItems(new_items)
		return m, nil
	case itemsMsg:
		for _, it := range msg {
			m.cache = append(m.cache, it.(item))
		}
		log.Print("Loaded assignments")
		m.list.StopSpinner()
		statusCmd := m.list.NewStatusMessage("Φόρτωση επιτυχής!")
		return m, tea.Batch(updateCmd, statusCmd)
	case errorMsg:
		log.Print(msg.err)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m listModel) View() string {
	return docStyle.Render(m.list.View())
}

type item struct {
	assignment assignment.Assignment
	selected   bool
}

func (i item) FilterValue() string { // TODO: keybinds to change this func to others
	return i.assignment.Course.Name
}

type itemDelegate struct{}

var (
	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			BorderLeft(true).
			BorderStyle(lipgloss.HiddenBorder()).
			Faint(true)
	itemTitleStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			BorderLeft(true).
			BorderStyle(lipgloss.HiddenBorder()).
			Bold(true)
	hoverItemStyle = itemStyle.Copy().
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#0b365b")).
			Foreground(lipgloss.Color("#6eaede"))
	hoverItemTitleStyle = itemTitleStyle.Copy().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#0b365b")).
				Foreground(lipgloss.Color("#6eaede"))
)

func (itemD itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var deadline string
	if !time.Time.Equal(i.assignment.Deadline, time.Time{}) { // HACK: how to check for uninialized .Deadline
		deadline = "Παράδωση εώς: " + i.assignment.Deadline.Format("02/01/2006 15:04:05")
	}

	item_style := itemStyle.Copy()
	title_style := itemTitleStyle.Copy()
	item_style.Underline(i.selected)
	title_style.Underline(i.selected)

	if index == m.Index() {
		item_style = hoverItemStyle
		title_style = hoverItemTitleStyle

	}

	fmt.Fprint(w, title_style.Render(i.assignment.Title)+"\n")      // title
	fmt.Fprint(w, item_style.Render(i.assignment.Course.Name)+"\n") // course
	fmt.Fprint(w, item_style.Render(deadline))                      // deadline

}

func (itemD itemDelegate) Height() int {
	return 3
}

func (itemD itemDelegate) Spacing() int {
	return 0
}

func (itemD itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		}
	}
	return nil
}

type itemsMsg []list.Item
type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

func getAssignments() tea.Msg {
	log.Print("Loading assignments from eclass..")
	opts, creds, err := config.Import()
	if err != nil {
		return errorMsg{err}
	}

	flags.Read(opts, creds)

	err = config.Ensure(opts, creds)
	if err != nil {
		return errorMsg{err}
	}

	a, err := assignment.Get(opts, creds)
	if err != nil {
		return errorMsg{err}
	}
	var items = make([]list.Item, len(a))

	for i, ass := range a {
		items[i] = item{
			assignment: ass,
		}
	}

	return itemsMsg(items)
}

func mockGetAssignments() tea.Msg {
	a := []assignment.Assignment{ // {{{
		{
			ID: "A1",
			Course: &course.Course{
				ID:   "CS101",
				Name: "Name 1",
				URL:  "https://some.random.url",
			},
			Title:    "Course #1",
			Deadline: time.Now(),
			IsSent:   false,
		},
		{
			ID: "A2",
			Course: &course.Course{
				ID:   "CS302",
				Name: "Name 2",
				URL:  "https://some.random.url",
			},
			Title:    "Course #2",
			Deadline: time.Now(),
			IsSent:   false,
		},
		{
			ID: "A3",
			Course: &course.Course{
				ID:   "CS404",
				Name: "Name 0",
				URL:  "https://some.random.url",
			},
			Title:    "Course #3",
			Deadline: time.Now(),
			IsSent:   false,
		},
	} // }}}

	var items = make([]list.Item, len(a))

	for i, ass := range a {
		items[i] = item{
			assignment: ass,
		}
	}

	return itemsMsg(items)
}
