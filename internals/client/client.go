package client

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	PORT = "4000"
	TYPE = "tcp"
)

var HOST string

type AppState int

const (
	LOGIN = iota
	CHAT
)

func writeToServer(chatMessage ChatMsg, conn *net.TCPConn) {
	_, err := conn.Write([]byte(chatMessage.message))
	if err != nil {
		log.Fatalf("Write data failed: %s", err)
	}
}

func readFromServer(conn *net.TCPConn, chatMessageChan chan ChatMsg) tea.Cmd {
	return func() tea.Msg {
		for {
			received := make([]byte, 1024)

			_, err := conn.Read(received)
			if err != nil {
				log.Printf("Read data failed: %s", err)
				os.Exit(1)
			}

			chatMessageChan <- ChatMsg{
				message: string(received),
			}
		}
	}
}

func waitForMessage(chatMessageChan chan ChatMsg) tea.Cmd {
	return func() tea.Msg {
		return <-chatMessageChan
	}
}

func Serve() *net.TCPConn {
	tcpServer, err := net.ResolveTCPAddr(TYPE, HOST+":"+PORT)
	if err != nil {
		log.Printf("ResolveTCPAddr failed: %s", err)
		os.Exit(1)
	}

	conn, err := net.DialTCP(TYPE, nil, tcpServer)
	if err != nil {
		log.Printf("Dial failed: %s", err)
		os.Exit(1)
	}

	// FIXME: Handle this
	// defer conn.Close()

	return conn
}

type ChatMsg struct {
	message string
}

func Render(ipAddr string) {
	HOST = ipAddr

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	textarea textarea.Model
	chat     chatModel
	appState AppState

	vh int
	vw int
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Provide an username..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 20

	ta.SetWidth(30)
	ta.SetHeight(1)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	return model{
		textarea: ta,
		appState: LOGIN,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.EnterAltScreen,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vh = msg.Height
		m.vw = msg.Width

		return m, tea.Batch(
			textarea.Blink,
			tea.EnterAltScreen,
		)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			username := m.textarea.Value()
			chatModel := initialChatModel(username, m.vh, m.vw)
			return chatModel, tea.Batch(
				textarea.Blink,
				readFromServer(chatModel.conn, chatModel.chatMessageChan),
				waitForMessage(chatModel.chatMessageChan),
			)
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	str := fmt.Sprintf(
		"%s\n",
		m.textarea.View(),
	) + "\n\n"

	horizontalPadding := (m.vw - lipgloss.Width(lipgloss.NewStyle().Render(str))) / 2
	verticalPadding := (m.vh - 1) / 2

	return lipgloss.NewStyle().
		Padding(verticalPadding, horizontalPadding).
		Render(str)
}

type chatModel struct {
	textarea textarea.Model
	err      error

	chatMessageChan chan ChatMsg
	conn            *net.TCPConn

	senderStyle lipgloss.Style
	username    string
	viewport    viewport.Model
	messages    []string
}

func initialChatModel(username string, vh int, vw int) chatModel {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(vw)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(vw, vh-ta.Height()-5)

	// FIXME: Not working :( But atleast it stopped the j and k thing
	vp.KeyMap.Up = key.NewBinding(
		key.WithKeys("pgup", "up"),        // actual keybindings
		key.WithHelp("↑/pgup", "move up"), // corresponding help text
	)

	vp.KeyMap.Down = key.NewBinding(
		key.WithKeys("pgdown", "down"),
		key.WithHelp("↓/pgdown", "move down"),
	)

	vp.SetContent("Welcome to the chat room!\nType a message and press Enter to send.")

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return chatModel{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("1")),

		username: strings.ReplaceAll(username, "\n", ""),

		conn: Serve(),

		chatMessageChan: make(chan ChatMsg),
	}
}

func (m chatModel) Init() tea.Cmd {
	return tea.Batch()
}

func (m chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			text := m.textarea.Value()

			go writeToServer(ChatMsg{message: m.username + ": " + text}, m.conn)

			m.messages = append(m.messages, m.senderStyle.Render("You: ")+text)
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()
		}
	case ChatMsg:
		m.messages = append(m.messages, msg.message)
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.textarea.Reset()
		m.viewport.GotoBottom()

		return m, tea.Batch(tiCmd, vpCmd, waitForMessage(m.chatMessageChan))
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m chatModel) View() string {
	str := fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"

	return str
}
