package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"assignments/assignment"
	"assignments/config"
	"assignments/course"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	lip "github.com/charmbracelet/lipgloss"
)

type listModel struct {
	list       list.Model
	cache      []item
	showHidden bool
	keys       keyBinds
	config     config.Config
}

func NewList(config config.Config) listModel {
	if config.Options.ExcludedAssignments == nil { // FIX: should't these be already made?
		config.Options.ExcludedAssignments = make(map[string][]string)
	}
	if config.Options.ExcludedCourses == nil {
		config.Options.ExcludedCourses = make(map[string]struct{})
	}

	m := listModel{
		list:       list.New([]list.Item{}, itemDelegate{}, 0, 0),
		showHidden: false,
		keys:       newKeyBinds(),
		config:     config,
	}
	m.list.Title = "Εργασίες"
	m.list.SetShowStatusBar(true)
	statusTime := time.Second * 2
	m.list.StatusMessageLifetime = statusTime
	m.list.SetStatusBarItemName("Εργασία", "Εργασίες")
	m.list.SetSpinner(spinner.Dot)
	// TODO: this looks like a bubbletea bug, spinner's style is unused in list/list.go
	// m.list.Styles.Spinner = lip.NewStyle().Background(uniwaOrange).Border(lip.DoubleBorder())
	m.list.StartSpinner()
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			m.keys.toggleHidden,
			m.keys.toggleHideCourse,
			m.keys.toggleHideAssignment,
			m.keys.saveConfig,
			m.keys.toggleIncludeExpired,
		}
	}
	return m
}

type keyBinds struct {
	toggleHideAssignment key.Binding
	toggleHideCourse     key.Binding
	toggleHidden         key.Binding
	saveConfig           key.Binding
	toggleIncludeExpired key.Binding
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
		saveConfig: key.NewBinding(
			key.WithKeys("s", "σ"),
			key.WithHelp("s|σ", "Αποθήκευση"),
		),
		toggleIncludeExpired: key.NewBinding(
			key.WithKeys("i", "ι"),
			key.WithHelp("i|ι", "Include expired"), // TODO: greek help description
		),
	}
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(
		m.list.StartSpinner(),
		getAssignments,
		mockGetAssignments,
        updateTitleCmd,
	)
}

var docStyle = lip.NewStyle().Margin(1, 2)

type updateTitleMsg struct{}

func updateTitleCmd() tea.Msg { return updateTitleMsg{} }

type writeConfigMsg struct{}

func writeConfigCmd() tea.Msg { return writeConfigMsg{} }

type updateMsg struct{}

func updateItemsCmd() tea.Msg { return updateMsg{} }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.saveConfig):
			err := config.Export(m.config.Options, m.config.Credentials)
			if err != nil {
				return m, errorCmd(err)
			}
			cmd := m.list.NewStatusMessage("Αποθηκέυση επιτυχής!")
			return m, cmd
		case key.Matches(msg, m.keys.toggleHideAssignment):
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				log.Print("Type Assertion failed")
			}
			// toggle excluded
			excluded := false
			for excluded_assignment_ID := range m.config.Options.ExcludedAssignments {
				if i.assignment.ID == excluded_assignment_ID {
					excluded = true
				}
			}
            var statusCmd tea.Cmd
			if excluded {
				delete(m.config.Options.ExcludedAssignments, i.assignment.ID)
                statusCmd = m.list.NewStatusMessage("Επανέφερες την εργασία " + i.assignment.Title + ".")
			} else {
				m.config.Options.ExcludedAssignments[i.assignment.ID] = []string{} // HACK: uhh what do we put here again?
                statusCmd = m.list.NewStatusMessage("Έκρυψες την εργασία " + i.assignment.Title + ".")
			}
			return m, tea.Batch(updateItemsCmd, statusCmd)
		case key.Matches(msg, m.keys.toggleHideCourse):
			i, ok := m.list.SelectedItem().(item)
			if !ok {
				log.Print("Type Assertion failed")
			}
			excluded := false
			for excluded_course_ID := range m.config.Options.ExcludedCourses {
				if i.assignment.Course.ID == excluded_course_ID {
					excluded = true
				}
			}
            var statusCmd tea.Cmd
			if excluded {
				delete(m.config.Options.ExcludedCourses, i.assignment.Course.ID)
                statusCmd = m.list.NewStatusMessage("Επανέφερες τις εργασίες του μαθήματος " + i.assignment.Course.Name + ".")
			} else {
				m.config.Options.ExcludedCourses[i.assignment.Course.ID] = struct{}{}
                statusCmd = m.list.NewStatusMessage("Έκρυψες τις εργασίες του μαθήματος " + i.assignment.Course.Name + ".")
			}
			return m, tea.Batch(updateItemsCmd, statusCmd)
		case key.Matches(msg, m.keys.toggleHidden):
			m.showHidden = !m.showHidden
			return m, tea.Batch(updateItemsCmd, updateTitleCmd)
		case key.Matches(msg, m.keys.toggleIncludeExpired):
			m.config.Options.IncludeExpired = !m.config.Options.IncludeExpired
			return m, tea.Batch(updateItemsCmd, updateTitleCmd)
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
			hide := false
			if item.shouldHideAssignment(m.config.Options.ExcludedAssignments) {
				hide = true
			}

			if item.shouldHideCourse(m.config.Options.ExcludedCourses) {
				hide = true
			}

			// if the deadline has passed and we don't include expired OR we show hidden
			// hide this item
			if item.shouldHideExpired() && (!m.config.Options.IncludeExpired || m.showHidden) {
				hide = true
			}

			// log.Println(item.assignment.ID, "hidden? ", hide, " because ", item.hideReason)
			if hide == m.showHidden {
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
		if len(msg) > 5 { // NOTE: mockGetAssignments returns 5, this is a hack
			m.list.StopSpinner()
		}
		statusCmd := m.list.NewStatusMessage("Φόρτωση επιτυχής!")
		return m, tea.Batch(updateItemsCmd, statusCmd)
	case updateTitleMsg:
		var title string
		if m.showHidden {
			title += "Κρυμμένες εργασίες"
		} else {
			title += "Εργασίες"
		}
		if m.config.Options.IncludeExpired {
			title += " | Εμφανίζονται Εκπρόθεσμες"
		}
		m.list.Title = title
		return m, nil
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
	hideReason string
}

func (i *item) shouldHideAssignment(excludedAssignments map[string][]string) bool {
	for hidden_ass := range excludedAssignments {
		if hidden_ass == i.assignment.ID {
			i.hideReason = "κρυμένη εργασία"
			return true
		}
	}
	return false
}

func (i *item) shouldHideCourse(excludedCourses map[string]struct{}) bool {
	for hidden_course := range excludedCourses {
		if hidden_course == i.assignment.Course.ID {
			i.hideReason = "εργασία κρυμένου μαθήματος"
			return true
		}
	}

	return false
}

func (i *item) shouldHideExpired() bool {
	if i.isExpired() {
		i.hideReason = "εκπρόθεσμη"
		return true
	}
	return false
}

func (i item) FilterValue() string { // TODO: keybinds to change this func to others
	return i.assignment.Title
}

func (i item) isExpired() bool {
	return i.assignment.Deadline.Before(time.Now())
}

var ( // {{{
	uniwaBlue      = lip.Color("#0b365b")
	hoverBg        = lip.Color("#030F27")
	uniwaLightBlue = lip.Color("#6eaede")
	uniwaOrange    = lip.Color("#e67c17")
	baseStyle      = lip.NewStyle().
			PaddingLeft(4).
			Foreground(lip.Color("#777777")).
			Faint(true)
	titleStyle = baseStyle.Copy().
			PaddingLeft(2).
			UnsetForeground().
			MaxWidth(47).
			Bold(true)
	lateStyle = baseStyle.Copy().
			Foreground(lip.Color("1"))
	doneStyle = baseStyle.Copy().
			Foreground(lip.Color("2"))
	normalStyle = lip.NewStyle().
			BorderStyle(lip.HiddenBorder()).
			BorderLeft(true)
	hoverStyle = lip.NewStyle().
			BorderStyle(lip.ThickBorder()).
			BorderLeft(true).
			BorderForeground(uniwaBlue).
			Foreground(uniwaLightBlue)
) // }}}

type itemDelegate struct{}

func (itemD itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var title, course, date, hideReason string

	title = titleStyle.Render(i.assignment.Title)
	course = baseStyle.Render(i.assignment.Course.Name)

	if i.assignment.IsSent {
		date = doneStyle.Render("Παραδώθηκε πριν της:")
	} else {
		if i.isExpired() {
			date = lateStyle.Render("Εκπρόθεσμη:         ")
		} else {
			date = baseStyle.Render("Παράδωση εώς:       ")
		}
	}

	date += baseStyle.Render(i.assignment.Deadline.Format("02/01/2006 15:04:05"))

	if i.hideReason != "" {
		hideReason = baseStyle.Render("Κρυμένη γιατί είναι " + i.hideReason)
	}

	result := lip.JoinVertical(lip.Left, title, date, course)
	result = lip.JoinHorizontal(lip.Top, result, hideReason)

	if index == m.Index() {
		result = hoverStyle.Render(result)
	} else {
		result = normalStyle.Render(result)
	}

	fmt.Fprint(w, result)
}

func (itemD itemDelegate) Height() int {
	return 3
}

func (itemD itemDelegate) Spacing() int {
	return 0
}

func (itemD itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

type itemsMsg []list.Item
type errorMsg struct{ err error }

func errorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{err}
	}
}

func (e errorMsg) Error() string { return e.err.Error() }

// getAssignments gets ALL the assignments from eclass,
// even excluded  ones, we filter them out later
func getAssignments() tea.Msg {
	log.Print("Loading assignments from eclass..")
	opts, creds, err := config.Import()
	if err != nil {
		return errorMsg{err}
	}

	err = config.Ensure(opts, creds)
	if err != nil {
		return errorMsg{err}
	}

	// get ALL assignments
	opts.ExcludedAssignments = make(map[string][]string)
	opts.ExcludedCourses = make(map[string]struct{})
    opts.IncludeExpired = true

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
			IsSent:   true,
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
			Deadline: time.Now().Add(time.Hour),
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
