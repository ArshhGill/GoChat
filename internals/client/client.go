package client

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	HOST = "127.0.0.1"
	PORT = "4000"
	TYPE = "tcp"
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

func Render() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	textarea textarea.Model
	err      error

	chatMessageChan chan ChatMsg
	conn            *net.TCPConn

	senderStyle lipgloss.Style
	username    string
	viewport    viewport.Model
	messages    []string
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)

	vp.SetContent(`type /connect {username} to connect to the server :)`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:    ta,
		messages:    []string{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch() // goes to fullscreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fmt.Println(msg.Height, m.textarea.Height())

		m.viewport.Height = msg.Height / 2
		m.viewport.Width = msg.Width

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
			if m.conn == nil {
				cmd := strings.Split(m.textarea.Value(), " ")

				if len(cmd) < 2 || cmd[0] != "/connect" {
					m.viewport.SetContent("Incorrect command. Type /connect {username} to connect to the server")
					m.textarea.Reset()

					return m, textarea.Blink
				}

				m.chatMessageChan = make(chan ChatMsg)
				m.conn = Serve()

				m.username = cmd[1]

				m.viewport.SetContent("Welcome to the chat room!\nType a message and press Enter to send.")

				m.textarea.Reset()

				return m, tea.Batch(
					textarea.Blink,
					readFromServer(m.conn, m.chatMessageChan),
					waitForMessage(m.chatMessageChan),
				)
			}

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

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n"
}
