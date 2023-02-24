package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Huray-hub/eclass-utils/assignments/assignment"
	// "github.com/Huray-hub/eclass-utils/assignments/calendar"
	"github.com/Huray-hub/eclass-utils/assignments/cmd/flags"
	// "github.com/Huray-hub/eclass-utils/assignments/cmd/output"
	"github.com/Huray-hub/eclass-utils/assignments/config"
	// "github.com/Huray-hub/eclass-utils/assignments/course"

	tea "github.com/charmbracelet/bubbletea"
	// "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

func init() {
	homeCache, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err.Error())
	}

	path := filepath.Join(homeCache, "eclass-utils")
	if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	file, err := os.OpenFile(
		filepath.Join(path, "assignments.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}
	// defer func() {
	// 	if err = file.Close(); err != nil {
	// 		log.Fatal(err.Error())
	// 	}
	// }()

	log.SetOutput(file)
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type listModel struct {
	list list.Model
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(
        m.list.StartSpinner(),
        getAssignments,
    )
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
    case itemsMsg:
        log.Print("Loaded assignments from eclass")
        cmd := m.list.SetItems(msg)
        m.list.StopSpinner()
        return m, cmd
    case itemsErrorMsg:
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

type item assignment.Assignment

func (i item) getDeadline() string {
	var deadline string

	if !time.Time.Equal(i.Deadline, time.Time{}) { // HACK: how to check for uninialized .Deadline
		deadline = "Deadline: " + i.Deadline.Format("02/01/2006 15:04:05")
	}
	return i.Course.Name + " " + deadline
}

func (i item) FilterValue() string { // TODO: keybinds to change this func to others
	return i.Course.Name
}

type itemDelegate struct{}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4).Faint(true)
	selectedItemStyle = itemStyle.Copy().Faint(false).Foreground(lipgloss.Color("2"))
	itemTitleStyle    = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("6"))
)

func (itemD itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var deadline string
	if !time.Time.Equal(i.Deadline, time.Time{}) { // HACK: how to check for uninialized .Deadline
		deadline = "Παράδωση εώς: " + i.Deadline.Format("02/01/2006 15:04:05")
	}

	fn := itemStyle.Render

	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render(s)
		}
	}

	fmt.Fprint(w, itemTitleStyle.Render(i.Title)+"\n") // title
	fmt.Fprint(w, fn(i.Course.Name)+"\n")              // course
	fmt.Fprint(w, fn(deadline))                        // deadline

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
type itemsErrorMsg struct{ err error }

func (e itemsErrorMsg) Error() string { return e.err.Error() }

func getAssignments() tea.Msg {
    log.Print("Loading assignments from eclass..")
	opts, creds, err := config.Import()
	if err != nil {
        return itemsErrorMsg{err}
	}

	flags.Read(opts, creds)

	err = config.Ensure(opts, creds)
	if err != nil {
        return itemsErrorMsg{err}
	}

	a, err := assignment.Get(opts, creds)
	if err != nil {
        return itemsErrorMsg{err}
	}
	// a := []assignment.Assignment{// {{{
	// 	{
	// 		ID: "A1",
	// 		Course: &course.Course{
	// 			ID:   "CS101",
	// 			Name: "Name 1",
	// 			URL:  "https://some.random.url",
	// 		},
	// 		Title:    "Course #1",
	// 		Deadline: time.Now(),
	// 		IsSent:   false,
	// 	},
	// 	{
	// 		ID: "A2",
	// 		Course: &course.Course{
	// 			ID:   "CS302",
	// 			Name: "Name 2",
	// 			URL:  "https://some.random.url",
	// 		},
	// 		Title:    "Course #2",
	// 		Deadline: time.Now(),
	// 		IsSent:   false,
	// 	},
	// 	{
	// 		ID: "A1",
	// 		Course: &course.Course{
	// 			ID:   "CS404",
	// 			Name: "Name 0",
	// 			URL:  "https://some.random.url",
	// 		},
	// 		Title:    "Course #3",
	// 		Deadline: time.Now(),
	// 		IsSent:   false,
	// 	},
	// }// }}}

    var items = make([]list.Item, len(a))

    for i, ass := range a {
        items[i] = item(ass)
    }

	return  itemsMsg(items)
}

func main() {

	m := listModel{list: list.New([]list.Item{}, itemDelegate{}, 0, 0)}
	m.list.Title = "Εργασίες"

	m.list.SetSpinner(spinner.Dot)
	m.list.Styles.Spinner = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	m.list.StartSpinner()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
