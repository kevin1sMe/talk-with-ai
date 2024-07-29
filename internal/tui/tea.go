package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"

	log "github.com/sirupsen/logrus"
)

var (
	// Color颜色表格： https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
	focusedStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("11"))
	blurredStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("7"))
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// 定义历史记录样式
	userStyle      = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("2")) // 绿色
	assistantStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6"))  // 红色
)

// 定义一个消息类型，用于在通道中传递的事件
type Event struct {
	Type    string
	Payload string
}

type ChatMessage struct {
	Role    string
	Content string
}

type model struct {
	modelList     list.Model
	toneList      list.Model
	emotionList   list.Model
	viewport      viewport.Model
	questionInput textinput.Model
	// pastQuestions  []string
	chatHistory    []ChatMessage
	currentFocus   int
	height         int
	width          int
	notification   string
	notificationCh chan string
	isRecording    bool
	processing     bool // 处理中，不允许再输入

	eventChan chan Event
	inChan    chan Event
	logger    *log.Logger
}

type toggleMsg struct{}

func InitialModel(l *log.Logger, out chan Event, in chan Event) model {
	modelItems := []list.Item{
		item{title: "yi-large", desc: "yi-large 模型"},
		item{title: "hunyuan", desc: "hunyuan 模型"},
		item{title: "gpt-4o", desc: "gpt-4o 模型"},
		// 添加更多模型选项
	}

	toneItems := []list.Item{
		item{title: "101016", desc: "智甜-女童声"},
		item{title: "101040", desc: "智川-四川女声"},
		item{title: "1009", desc: "智芸-知性女声"},
		item{title: "101019", desc: "智彤-粤语女声"},
		// 添加更多音色选项
	}

	emotionItems := []list.Item{
		item{title: "neutral", desc: "中性"},
		item{title: "angry", desc: "生气"},
		item{title: "exciting", desc: "兴奋"},
		item{title: "amaze", desc: "震惊"},
		// 添加更多情感选项
	}

	questionInput := textinput.New()
	questionInput.Placeholder = "在此输入问题..."
	questionInput.Focus()

	return model{
		modelList:      list.New(modelItems, list.NewDefaultDelegate(), 0, 0),
		toneList:       list.New(toneItems, list.NewDefaultDelegate(), 0, 0),
		emotionList:    list.New(emotionItems, list.NewDefaultDelegate(), 0, 0),
		viewport:       viewport.Model{},
		questionInput:  questionInput,
		currentFocus:   4, // 先默认选中输入框
		notificationCh: make(chan string, 1),
		isRecording:    false,
		processing:     false,
		eventChan:      out,
		inChan:         in,
		logger:         l,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.listenForNotification(), m.waitForInEvent())
}

func (m model) listenForNotification() tea.Cmd {
	return func() tea.Msg {
		return notificationMsg(<-m.notificationCh)
	}
}

func (m model) waitForInEvent() tea.Cmd {
	return func() tea.Msg {
		return eventMsg(<-m.inChan)
	}
}

type notificationMsg string
type eventMsg Event

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			close(m.eventChan)
			return m, tea.Quit
		case "tab":
			m.currentFocus = (m.currentFocus + 1) % 5
			if m.currentFocus == 4 {
				m.questionInput.Focus()
			} else {
				m.questionInput.Blur()
			}
		case "shift+tab":
			m.currentFocus = (m.currentFocus - 1 + 5) % 5
			if m.currentFocus == 4 {
				m.questionInput.Focus()
			} else {
				m.questionInput.Blur()
			}
		case "up":
			if m.currentFocus == 3 {
				m.viewport.LineUp(1)
			}
		case "down":
			if m.currentFocus == 3 {
				m.viewport.LineDown(1)
			}
		case "enter":
			switch m.currentFocus {
			case 0:
				selectedModel := m.modelList.SelectedItem().(item)
				m.notificationCh <- fmt.Sprintf("选择了模型: %s", selectedModel.Title())
				m.eventChan <- Event{Type: "model", Payload: selectedModel.Title()}
			case 1:
				selectedTone := m.toneList.SelectedItem().(item)
				m.notificationCh <- fmt.Sprintf("选择了音色: %s", selectedTone.Title())
				m.eventChan <- Event{Type: "tone", Payload: selectedTone.Title()}
			case 2:
				selectedEmotion := m.emotionList.SelectedItem().(item)
				m.notificationCh <- fmt.Sprintf("选择了情感: %s", selectedEmotion.Title())
				m.eventChan <- Event{Type: "emotion", Payload: selectedEmotion.Title()}
			case 3:
				log.Debug("选择了历史记录框")
				m.notificationCh <- "选择了历史记录"
			case 4:
				question := m.questionInput.Value()
				log.Debug("问题输入完毕", question)
				m.questionInput.SetValue("")
				m.notificationCh <- fmt.Sprintf("输入了问题: %s", question)
				m.eventChan <- Event{Type: "question", Payload: question}
			}
		default:
		}
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonWheelUp {
			m.viewport.LineUp(3)
			m.viewport, _ = m.viewport.Update(msg)
		} else if msg.Button == tea.MouseButtonWheelDown {
			m.viewport.LineDown(3)
			m.viewport, _ = m.viewport.Update(msg)
		} else if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			log.Debugf("按下鼠标左键: %v", msg)
			log.Debugf("MouseMsg: %v, (%v,%v)", msg, msg.X, msg.Y)
			// 检查是否按到了输入框内
			qX := m.width/5 + 2
			qY := m.viewport.Height + 2
			qW := m.viewport.Width
			qH := m.height/4 - 6

			log.Debugf("Pos:(%v,%v) width:%v, height:%v", qX, qY, qW, qH)

			if msg.X >= qX && msg.X <= qX+qW && msg.Y >= qY && msg.Y <= qY+qH {
				// 鼠标点击了问题输入框
				// m.questionInput.Focus()
				log.Debug("点中了问题输入框")
				return m, toggleRecording
			} else {
				log.Debug("没点中问题输入框")
			}
		} else if msg.Button == tea.MouseButtonRight {
			log.Debugf("按下鼠标右键: %v", msg)
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		listHeight := m.height/4 - 2 // 减去边框的高度
		listWidth := m.width/5 - 2   // 减去边框的宽度

		m.modelList.SetHeight(listHeight)
		m.modelList.SetWidth(listWidth)

		m.toneList.SetHeight(listHeight)
		m.toneList.SetWidth(listWidth)

		m.emotionList.SetHeight(listHeight)
		m.emotionList.SetWidth(listWidth)

		m.viewport.Width = m.width*4/5 - 2
		m.viewport.Height = m.height*3/4 - 2 // 设置聊天历史的高度为窗口高度的一半
		m.viewport.SetContent(m.renderChatHistory(m.viewport.Width))
	case notificationMsg:
		m.notification = string(msg)
		return m, tea.Batch(m.listenForNotification(), m.clearNotification(), m.waitForInEvent())
	case eventMsg:
		log.Debugf("eventMsg: %v", msg)
		if msg.Type != "history" {
			break
		}

		var history []openai.ChatCompletionMessage
		err := json.Unmarshal([]byte(msg.Payload), &history)
		if err != nil {
			log.Errorf("Failed to unmarshal history: %v", err)
		} else {
			m.chatHistory = make([]ChatMessage, len(history))
			for i, msg := range history {
				m.chatHistory[i] = ChatMessage{
					Role:    string(msg.Role),
					Content: msg.Content,
				}
			}
		}
		m.viewport.SetContent(m.renderChatHistory(m.viewport.Width))
		return m, tea.Batch(m.listenForNotification(), m.clearNotification(), m.waitForInEvent())
	case toggleMsg:
		m.isRecording = !m.isRecording
		if m.isRecording {
			m.startRecording()
		} else {
			m.stopRecording()
		}
	}

	switch m.currentFocus {
	case 0:
		m.modelList, _ = m.modelList.Update(msg)
	case 1:
		m.toneList, _ = m.toneList.Update(msg)
	case 2:
		m.emotionList, _ = m.emotionList.Update(msg)
	case 4:
		m.questionInput, _ = m.questionInput.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m model) clearNotification() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return notificationMsg("")
	})
}
func (m model) View() string {
	// log.Debugf("View, height: %d, width: %d, currentFocus:%v\n", m.height, m.width, m.currentFocus)
	// 左边三个设置项
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderList("模型选择", m.modelList, 0),
		m.renderList("音色选择", m.toneList, 1),
		m.renderList("情感选择", m.emotionList, 2),
	)
	// 右边，下面，是输入框
	inputWidth := m.viewport.Width
	inputHeight := m.height/4 - 6 // 4个边框 + 最后2行
	// log.Debugf("View, height: %d, width: %d, viewport:(%v,%v) input:(%v,%v)\n", m.height, m.width,m.viewport.Width, m.viewport.Height, inputWidth, inputHeight)

	if m.isRecording {
		m.questionInput.Placeholder = "正在录音中，再次点击结束录音..."
		m.questionInput.Focus()
	} else {
		m.questionInput.Placeholder = "请输入..."
	}

	m.viewport.SetContent(m.renderChatHistory(m.viewport.Width))
	viewRender := blurredStyle.Render("聊天历史\n" + m.viewport.View())
	if m.currentFocus == 3 {
		viewRender = focusedStyle.Render("聊天历史\n" + m.viewport.View())
	}

	rightColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		viewRender,
		focusedStyle.
			Width(inputWidth).
			Height(inputHeight).
			Align(lipgloss.Left).
			Render(m.questionInput.View()),
	)

	ui := lipgloss.JoinHorizontal(lipgloss.Left, leftColumn, rightColumn)
	notification := ""
	if m.notification != "" {
		notification = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(m.notification)
	}
	return ui + "\n" + notification + "\n" + helpStyle.Render("按 Tab 切换焦点 • 按 q 退出")
}

func (m model) renderList(title string, l list.Model, index int) string {
	if m.currentFocus == index {
		return focusedStyle.Render(fmt.Sprintf("%s\n%s", title, l.View()))
	}
	return blurredStyle.Render(fmt.Sprintf("%s\n%s", title, l.View()))
}

func (m *model) renderChatHistory(width int) string {
	var chatContent strings.Builder
	textWidth := width*4/5 - 4 // 减去边框的宽度
	// log.Debugf("renderChatHistory, width:%v, textWidth:%v", width, textWidth)

	for _, msg := range m.chatHistory {
		var content string
		wrappedContent := WrapWords(msg.Content, textWidth)
		if msg.Role == "user" {
			content = userStyle.
				Align(lipgloss.Left).
				Render(wrappedContent)
		} else {
			content = assistantStyle.
				Align(lipgloss.Left).
				// Align(lipgloss.Right). // 左右为难，文本的对齐和边框都是这个？
				// PaddingLeft(width / 5).
				MarginLeft(width / 5).
				Render(wrappedContent)
		}
		chatContent.WriteString(content + "\n")
	}

	return chatContent.String()
}

func (m model) startRecording() {
	m.eventChan <- Event{Type: "audio_start", Payload: ""}
	m.notificationCh <- "开始录音"
	log.Debug("开始录音,发送开始事件")
}

func (m model) stopRecording() {
	m.eventChan <- Event{Type: "audio_stop", Payload: ""}
	m.notificationCh <- "结束录音"
	log.Debug("停止录音,发送停止事件")
}

func toggleRecording() tea.Msg {
	return toggleMsg{}
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }
