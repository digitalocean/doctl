package pager

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/armon/circbuf"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/digitalocean/doctl/commands/charm"
	"github.com/digitalocean/doctl/commands/charm/template"
	"github.com/digitalocean/doctl/commands/charm/text"
)

type WriterStringer interface {
	io.Writer
	fmt.Stringer
}

type Pager struct {
	title        string
	titleSpinner bool
	bufSize      int64
	buffer       WriterStringer
	prog         *tea.Program
	model        *pagerModel
	exited       bool
}

type Option func(*Pager)

func New(opts ...Option) (*Pager, error) {
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

func WithBufferSize(size int64) Option {
	return func(p *Pager) {
		p.bufSize = size
	}
}

func WithTitle(title string) Option {
	return func(p *Pager) {
		p.title = title
	}
}

func WithTitleSpinner(spinner bool) Option {
	return func(p *Pager) {
		p.titleSpinner = spinner
	}
}

func (p *Pager) Write(b []byte) (int, error) {
	if p.exited {
		return os.Stdout.Write(b)
	}
	n, err := p.buffer.Write(b)
	if p.prog != nil {
		p.prog.Send(msgUpdate{})
	}
	return n, err
}

func (p *Pager) Start(ctx context.Context) error {
	p.model = newPagerModel(ctx, p.buffer, p.title, p.titleSpinner)
	prog := tea.NewProgram(
		p.model,
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	p.prog = prog

	err := prog.Start()
	p.exited = true
	p.prog = nil

	fmt.Fprint(charm.Indent(4), p.buffer.String())
	if err != nil {
		return err
	} else if p.model.userCanceled {
		return charm.ErrCanceled
	}

	return nil
}

type pagerModel struct {
	ctx    context.Context
	cancel context.CancelFunc

	start        time.Time
	title        string
	buffer       WriterStringer
	ready        bool
	viewport     viewport.Model
	userCanceled bool
	spinner      *spinner.Model
}

func newPagerModel(ctx context.Context, buffer WriterStringer, title string, titleSpinner bool) *pagerModel {
	m := &pagerModel{
		buffer: buffer,
		title:  title,
		start:  time.Now(),
	}
	m.ctx, m.cancel = context.WithCancel(ctx)

	if titleSpinner {
		s := spinner.New(
			spinner.WithStyle(text.Muted.Lipgloss()),
			spinner.WithSpinner(spinner.Dot),
		)
		m.spinner = &s
	}

	return m
}

type msgQuit struct{}
type msgUpdate struct{}
type msgTick struct{}

func (m *pagerModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.timerTick(),
		func() tea.Msg {
			<-m.ctx.Done()
			return msgQuit{}
		},
	}

	if m.spinner != nil {
		cmds = append(cmds, m.spinner.Tick)
	}

	return tea.Batch(cmds...)
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
			template.Buffered(
				m.buffer,
				`{{nl}}{{error (print crossmark " got ctrl-c, cancelling. hit ctrl-c again to force exit.")}}{{nl}}`,
				nil,
			)
			m.userCanceled = true
			m.cancel()

			// we don't need to do anything special to handle the second ctrl-c. the pager exits fairly quickly and once
			// that's complete nothing will be intercepting interrupt syscalls and so the second ctrl-c will go directly to
			// the go runtime.
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
	case spinner.TickMsg:
		sp, cmd := m.spinner.Update(msg)
		m.spinner = &sp
		cmds = append(cmds, cmd)
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *pagerModel) View() string {
	if !m.ready {
		return "\n  loading..."
	}
	return fmt.Sprintf("%s\n%s%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m *pagerModel) headerView() string {
	// title line
	title := m.title
	if m.spinner != nil {
		title = m.spinner.View() + title
	}
	title = lipgloss.NewStyle().Padding(1).PaddingBottom(0).Render(title)

	// elapsed time + horizontal divider line
	elapsed := fmt.Sprintf(
		"%s%s%s",
		text.Muted.S("───["),
		time.Since(m.start).Truncate(time.Second),
		text.Muted.S("]"),
	)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(elapsed)))
	line = text.Muted.S(line)

	return fmt.Sprintf("%s\n%s%s\n", title, elapsed, line)
}

func (m *pagerModel) footerView() string {
	if m.viewport.AtBottom() {
		return ""
	}

	info := "┤ scroll down for new logs "
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return "\n" + text.Highlight.S(line+info)
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
