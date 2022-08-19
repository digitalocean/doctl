package charm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/armon/circbuf"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WriterStringer interface {
	io.Writer
	fmt.Stringer
}

type Pager struct {
	title   string
	bufSize int64
	buffer  WriterStringer
	prog    *tea.Program
	model   *pagerModel
}

type PagerOpt func(*Pager)

func NewPager(opts ...PagerOpt) (*Pager, error) {
	p := &Pager{
		title:   "Output",
		bufSize: 3 << (10 * 2), // 3MB
	}
	for _, opt := range opts {
		opt(p)
	}
	var err error
	p.buffer, err = circbuf.NewBuffer(p.bufSize)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func PagerWithBufferSize(size int64) PagerOpt {
	return func(p *Pager) {
		p.bufSize = size
	}
}

func PagerWithTitle(title string) PagerOpt {
	return func(p *Pager) {
		p.title = title
	}
}

func (p *Pager) Write(b []byte) (int, error) {
	n, err := p.buffer.Write(b)
	p.prog.Send(msgUpdate{})
	return n, err
}

func (p *Pager) Start(ctx context.Context) error {
	p.model = newPagerModel(ctx, p.buffer, p.title)
	prog := tea.NewProgram(
		p.model,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	p.prog = prog

	err := prog.Start()
	fmt.Fprintln(Indent(4), p.buffer.String())
	return err
}

type pagerModel struct {
	ctx    context.Context
	cancel context.CancelFunc

	start    time.Time
	title    string
	buffer   WriterStringer
	ready    bool
	viewport viewport.Model
}

func newPagerModel(ctx context.Context, buffer WriterStringer, title string) *pagerModel {
	m := &pagerModel{
		buffer: buffer,
		title:  title,
		start:  time.Now(),
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	return m
}

type msgQuit struct{}
type msgUpdate struct{}
type msgTick struct{}

func (m *pagerModel) Init() tea.Cmd {
	return tea.Batch(
		m.timerTick(),
		func() tea.Msg {
			<-m.ctx.Done()
			return msgQuit{}
		},
	)
}

func (m *pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case msgQuit:
		return m, tea.Quit

	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			TemplateBuffered(
				m.buffer,
				`{{nl}}{{error (join " " crossmark "got ctrl-c, cancelling build")}}{{nl}}`,
				nil,
			)
			m.cancel()
			return m, nil
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.buffer.String())
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	case msgUpdate:
		m.viewport.SetContent(m.buffer.String())
		m.viewport.GotoBottom()
	case msgTick:
		if m.ctx.Err() == nil {
			cmds = append(cmds, m.timerTick())
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *pagerModel) View() string {
	if !m.ready {
		return "\n  Loading..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m *pagerModel) headerView() string {
	title := m.titleStyle().Render(m.title)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	line = lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B")).Render(line)
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *pagerModel) footerView() string {
	info := m.infoStyle().Render(time.Since(m.start).Truncate(time.Second).String())
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	line = lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B")).Render(line)
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *pagerModel) titleStyle() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Right = "├"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#dddddd")).
		BorderStyle(b).
		BorderForeground(lipgloss.Color("#9B9B9B")).
		Padding(0, 1)
}

func (m *pagerModel) infoStyle() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Left = "┤"
	return m.titleStyle().Copy().BorderStyle(b).Foreground(lipgloss.Color("#dddddd"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p *pagerModel) timerTick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgTick{}
	})
}
