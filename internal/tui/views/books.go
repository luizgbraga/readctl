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


type Book struct {
	storage.Book
	TopicCount int
}

func (b Book) FilterValue() string { return b.Book.Title }


type booksLoadedMsg struct {
	books []Book
}

type bookCreatedMsg struct {
	bookID int
}

type bookDeletedMsg struct{}

type SelectedBookMsg struct {
	Book storage.Book
}


type BooksModel struct {
	list        list.Model
	db          *sql.DB
	width       int
	height      int
	creating    bool
	titleInput  textinput.Model
	authorInput textinput.Model
	focusedIdx  int // 0 = title, 1 = author
	deleting    bool
	err         error
}

// bookDelegate renders book items in the list
type bookDelegate struct{}

func (d bookDelegate) Height() int                             { return 2 }
func (d bookDelegate) Spacing() int                            { return 1 }
func (d bookDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d bookDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	book, ok := item.(Book)
	if !ok {
		return
	}

	title := book.Book.Title
	desc := fmt.Sprintf("%s • %d topics", book.Author, book.TopicCount)

	isSelected := index == m.Index()

	var titleStyle, descStyle lipgloss.Style
	if isSelected {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
		descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c6c6c"))
	}

	output := titleStyle.Render(title) + "\n" + descStyle.Render(desc)
	fmt.Fprint(w, output)
}


func NewBooksModel(db *sql.DB) BooksModel {
	// Create list with custom delegate
	delegate := bookDelegate{}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Books"

	// Create text inputs for the form
	titleInput := textinput.New()
	titleInput.Placeholder = "Enter book title"
	titleInput.CharLimit = 200

	authorInput := textinput.New()
	authorInput.Placeholder = "Enter author name"
	authorInput.CharLimit = 100

	return BooksModel{
		list:        l,
		db:          db,
		titleInput:  titleInput,
		authorInput: authorInput,
	}
}


func (m BooksModel) Init() tea.Cmd {
	return m.loadBooksCmd()
}


func (m BooksModel) Update(msg tea.Msg) (BooksModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for header/footer
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case booksLoadedMsg:
		items := make([]list.Item, len(msg.books))
		for i, book := range msg.books {
			items[i] = book
		}
		m.list.SetItems(items)
		return m, nil

	case bookCreatedMsg:
		m.creating = false
		m.titleInput.SetValue("")
		m.authorInput.SetValue("")
		m.focusedIdx = 0
		return m, m.loadBooksCmd()

	case bookDeletedMsg:
		m.deleting = false
		return m, m.loadBooksCmd()

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
			m.titleInput.Focus()
			m.focusedIdx = 0
			return m, textinput.Blink

		case "d":
			if len(m.list.Items()) > 0 {
				m.deleting = true
				return m, nil
			}
			return m, nil

		case "enter":
			if i := m.list.Index(); i >= 0 && i < len(m.list.Items()) {
				if book, ok := m.list.SelectedItem().(Book); ok {
					return m, func() tea.Msg {
						return SelectedBookMsg{Book: book.Book}
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

// handleCreateForm manages the book creation form
func (m BooksModel) handleCreateForm(msg tea.KeyMsg) (BooksModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.creating = false
		m.titleInput.SetValue("")
		m.authorInput.SetValue("")
		m.focusedIdx = 0
		return m, nil

	case "tab", "shift+tab":
		// Toggle focus between inputs
		if m.focusedIdx == 0 {
			m.focusedIdx = 1
			m.titleInput.Blur()
			m.authorInput.Focus()
			return m, textinput.Blink
		} else {
			m.focusedIdx = 0
			m.authorInput.Blur()
			m.titleInput.Focus()
			return m, textinput.Blink
		}

	case "enter":
		title := m.titleInput.Value()
		author := m.authorInput.Value()
		if title != "" && author != "" {
			return m, m.createBookCmd(title, author)
		}
		return m, nil
	}

	// Update the focused input
	var cmd tea.Cmd
	if m.focusedIdx == 0 {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.authorInput, cmd = m.authorInput.Update(msg)
	}
	return m, cmd
}


func (m BooksModel) View() string {
	if m.creating {
		return m.renderCreateForm()
	}

	// Empty state
	if len(m.list.Items()) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.SubtleColor).
			MarginTop(2).
			MarginLeft(2)
		return emptyStyle.Render("No books yet. Press n to add your first book.")
	}

	return m.list.View()
}

// renderCreateForm shows the book creation form
func (m BooksModel) renderCreateForm() string {
	formStyle := lipgloss.NewStyle().
		Padding(1, 2).
		MarginTop(2).
		MarginLeft(2)

	titleLabel := styles.HelpKeyStyle.Render("Title:")
	authorLabel := styles.HelpKeyStyle.Render("Author:")
	hint := styles.HelpDescStyle.Render("tab: next field • enter: save • esc: cancel")

	form := fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n\n%s",
		titleLabel,
		m.titleInput.View(),
		authorLabel,
		m.authorInput.View(),
		hint,
	)

	return formStyle.Render(form)
}


func (m BooksModel) IsDeleting() bool {
	return m.deleting
}


func (m BooksModel) IsCreating() bool {
	return m.creating
}


func (m *BooksModel) CancelDelete() {
	m.deleting = false
}


func (m BooksModel) SelectedBook() *storage.Book {
	if i := m.list.Index(); i >= 0 && i < len(m.list.Items()) {
		if book, ok := m.list.SelectedItem().(Book); ok {
			return &book.Book
		}
	}
	return nil
}


func (m *BooksModel) DeleteSelected() tea.Cmd {
	m.deleting = false
	if book := m.SelectedBook(); book != nil {
		return m.deleteBookCmd(book.ID)
	}
	return nil
}



func (m BooksModel) loadBooksCmd() tea.Cmd {
	return func() tea.Msg {
		storageBooks, err := storage.GetBooks(m.db)
		if err != nil {
			// In production, would return error message
			return booksLoadedMsg{books: []Book{}}
		}

		// Get topic counts for each book
		books := make([]Book, len(storageBooks))
		for i, sb := range storageBooks {
			topics, err := storage.GetTopics(m.db, sb.ID)
			topicCount := 0
			if err == nil {
				topicCount = len(topics)
			}
			books[i] = Book{
				Book:       sb,
				TopicCount: topicCount,
			}
		}

		return booksLoadedMsg{books: books}
	}
}

func (m BooksModel) createBookCmd(title, author string) tea.Cmd {
	return func() tea.Msg {
		id, err := storage.CreateBook(m.db, title, author)
		if err != nil {
			// In production, would return error message
			return nil
		}
		return bookCreatedMsg{bookID: id}
	}
}

func (m BooksModel) deleteBookCmd(bookID int) tea.Cmd {
	return func() tea.Msg {
		err := storage.DeleteBook(m.db, bookID)
		if err != nil {
			// In production, would return error message
			return nil
		}
		return bookDeletedMsg{}
	}
}
