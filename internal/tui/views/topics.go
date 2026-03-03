package views

import (
	"database/sql"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/luizgbraga/readctl/internal/storage"
	"github.com/luizgbraga/readctl/internal/tui/styles"
)


type Topic struct {
	Topic        storage.Topic
	FirstMessage string // First user message from chat
}

func (t Topic) FilterValue() string { return t.Topic.Name }


type topicsLoadedMsg struct {
	topics []Topic
}

type topicCreatedMsg struct {
	topicID int
	uuid    string
}

type topicDeletedMsg struct{}

type SelectedTopicMsg struct {
	Topic storage.Topic
}

type BackToBooksMsg struct{}

type ViewDocumentMsg struct {
	Topic storage.Topic
}


type TopicsModel struct {
	list      list.Model
	db        *sql.DB
	bookID    int
	bookTitle string
	width     int
	height    int
	creating  bool
	nameInput textinput.Model
	deleting  bool
	err       error
}

// topicDelegate renders topic items in the list
type topicDelegate struct{}

func (d topicDelegate) Height() int                             { return 2 }
func (d topicDelegate) Spacing() int                            { return 1 }
func (d topicDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d topicDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	topic, ok := item.(Topic)
	if !ok {
		return
	}

	// Title styling
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c6c6c"))

	if index == m.Index() {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#04B575")).Bold(true)
	}

	title := titleStyle.Render(topic.Topic.Name)

	// Show subtitle if messages exist
	subtitle := ""
	if topic.FirstMessage != "" {
		// Truncate if longer than 60 chars
		msg := topic.FirstMessage
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		subtitle = "\n" + subtitleStyle.Render(msg)
	}

	fmt.Fprint(w, title+subtitle)
}


func NewTopicsModel(db *sql.DB, bookID int, bookTitle string) TopicsModel {
	// Create list with custom delegate
	delegate := topicDelegate{}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Topics"

	// Create text input for the form
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter topic name"
	nameInput.CharLimit = 200

	return TopicsModel{
		list:      l,
		db:        db,
		bookID:    bookID,
		bookTitle: bookTitle,
		nameInput: nameInput,
	}
}


func (m TopicsModel) Init() tea.Cmd {
	return m.loadTopicsCmd()
}


func (m *TopicsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Reserve space for header/footer
	m.list.SetSize(width, height-4)
}


func (m TopicsModel) Update(msg tea.Msg) (TopicsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for header/footer
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case topicsLoadedMsg:
		items := make([]list.Item, len(msg.topics))
		for i, topic := range msg.topics {
			items[i] = topic
		}
		m.list.SetItems(items)
		return m, nil

	case topicCreatedMsg:
		m.creating = false
		m.nameInput.SetValue("")
		return m, m.loadTopicsCmd()

	case topicDeletedMsg:
		m.deleting = false
		return m, m.loadTopicsCmd()

	case tea.KeyMsg:
		// Handle creation form
		if m.creating {
			return m.handleCreateForm(msg)
		}

		// Handle delete confirmation
		if m.deleting {
			return m, nil // Parent (app.go) handles modal
		}

		// Handle normal list navigation
		switch msg.String() {
		case "n":
			m.creating = true
			m.nameInput.Focus()
			return m, textinput.Blink

		case "d":
			if len(m.list.Items()) > 0 {
				m.deleting = true
				return m, nil
			}
			return m, nil

		case "esc":
			return m, func() tea.Msg {
				return BackToBooksMsg{}
			}

		case "y":
			if i := m.list.Index(); i >= 0 && i < len(m.list.Items()) {
				if topic, ok := m.list.SelectedItem().(Topic); ok {
					return m, func() tea.Msg {
						return ViewDocumentMsg{Topic: topic.Topic}
					}
				}
			}
			return m, nil

		case "enter":
			if i := m.list.Index(); i >= 0 && i < len(m.list.Items()) {
				if topic, ok := m.list.SelectedItem().(Topic); ok {
					return m, func() tea.Msg {
						return SelectedTopicMsg{Topic: topic.Topic}
					}
				}
			}
			return m, nil
		}
	}

	// Delegate to list if not creating
	if !m.creating {
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleCreateForm manages the topic creation form
func (m TopicsModel) handleCreateForm(msg tea.KeyMsg) (TopicsModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.creating = false
		m.nameInput.SetValue("")
		return m, nil

	case "enter":
		name := m.nameInput.Value()
		if name != "" {
			return m, m.createTopicCmd(name)
		}
		return m, nil
	}

	// Update the input
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}


func (m TopicsModel) View() string {
	if m.creating {
		return m.renderCreateForm()
	}

	// Empty state
	if len(m.list.Items()) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.SubtleColor).
			MarginTop(2).
			MarginLeft(2)
		return emptyStyle.Render(fmt.Sprintf("No topics yet. Press n to create your first topic for %s.", m.bookTitle))
	}

	return m.list.View()
}

// renderCreateForm shows the topic creation form
func (m TopicsModel) renderCreateForm() string {
	formStyle := lipgloss.NewStyle().
		Padding(1, 2).
		MarginTop(2).
		MarginLeft(2)

	nameLabel := styles.HelpKeyStyle.Render("Topic Name:")
	hint := styles.HelpDescStyle.Render("enter: save • esc: cancel")

	form := fmt.Sprintf(
		"%s\n%s\n\n%s",
		nameLabel,
		m.nameInput.View(),
		hint,
	)

	return formStyle.Render(form)
}

// IsDeleting returns whether the view is in deleting mode
func (m TopicsModel) IsDeleting() bool {
	return m.deleting
}

// IsCreating returns whether the view is in creating mode (form active)
func (m TopicsModel) IsCreating() bool {
	return m.creating
}

// CancelDelete cancels the delete operation
func (m *TopicsModel) CancelDelete() {
	m.deleting = false
}

// SelectedTopic returns the currently selected topic
func (m TopicsModel) SelectedTopic() *storage.Topic {
	if i := m.list.Index(); i >= 0 && i < len(m.list.Items()) {
		if topic, ok := m.list.SelectedItem().(Topic); ok {
			return &topic.Topic
		}
	}
	return nil
}

// DeleteSelected deletes the currently selected topic
func (m *TopicsModel) DeleteSelected() tea.Cmd {
	m.deleting = false
	if topic := m.SelectedTopic(); topic != nil {
		return m.deleteTopicCmd(topic.ID)
	}
	return nil
}

// Commands (wrap storage operations in tea.Cmd)

func (m TopicsModel) loadTopicsCmd() tea.Cmd {
	return func() tea.Msg {
		storageTopics, err := storage.GetTopics(m.db, m.bookID)
		if err != nil {
			// In production, would return error message
			return topicsLoadedMsg{topics: []Topic{}}
		}

		topics := make([]Topic, len(storageTopics))
		for i, st := range storageTopics {
			// Load chat to get first user message
			messages, err := storage.LoadChat(st.ChatFileUUID)
			firstMsg := ""
			if err == nil && len(messages) > 0 {
				// Find first user message
				for _, msg := range messages {
					if msg.Role == "user" {
						firstMsg = msg.Content
						break
					}
				}
			}

			topics[i] = Topic{
				Topic:        st,
				FirstMessage: firstMsg,
			}
		}

		return topicsLoadedMsg{topics: topics}
	}
}

func (m TopicsModel) createTopicCmd(name string) tea.Cmd {
	return func() tea.Msg {
		id, uuid, err := storage.CreateTopic(m.db, m.bookID, name, "scholar")
		if err != nil {
			// In production, would return error message
			return nil
		}
		return topicCreatedMsg{topicID: id, uuid: uuid}
	}
}

func (m TopicsModel) deleteTopicCmd(topicID int) tea.Cmd {
	return func() tea.Msg {
		err := storage.DeleteTopic(m.db, topicID)
		if err != nil {
			// In production, would return error message
			return nil
		}
		return topicDeletedMsg{}
	}
}
