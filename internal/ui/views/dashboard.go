package views

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gh "github.com/jhermoso/ghtui/internal/github"
	"github.com/jhermoso/ghtui/internal/model"
)

// ─── focus states ─────────────────────────────────────────────────────────────

type dashFocus int

const (
	focusFilter dashFocus = iota
	focusList
	focusDetail
	focusEdit
	focusStatusPicker
)

// ─── filter fields ────────────────────────────────────────────────────────────

type filterFieldID int

const (
	ffOrg filterFieldID = iota
	ffProject
	ffIteration
	ffStatus
	ffType      // row 2 col 3 — must come before Assignee so iota order matches visual tab order
	ffAssignee
	ffLabel
	ffTitle
	ffDesc
	ffCount
)

// filterLabels are the display labels shown in the filter bar.
// The digit prefix is the quick-jump key shown to the user.
// Numbers follow visual reading order: left-to-right, top-to-bottom across all 3 rows.
// Row1: Org(1) Project(2)
// Row2: Iteration(3) Status(4) Type(5)
// Row3: Assignee(6) Label(7) Title(8) Desc(9)
var filterLabels = map[filterFieldID]string{
	ffOrg:       "1 Org",
	ffProject:   "2 Project",
	ffIteration: "3 Iteration",
	ffStatus:    "4 Status",
	ffType:      "5 Type",
	ffAssignee:  "6 Assignee",
	ffLabel:     "7 Label",
	ffTitle:     "8 Title",
	ffDesc:      "9 Desc",
}

// filterKey maps a digit key to the corresponding filter field.
var filterKey = map[string]filterFieldID{
	"1": ffOrg,
	"2": ffProject,
	"3": ffIteration,
	"4": ffStatus,
	"5": ffType,
	"6": ffAssignee,
	"7": ffLabel,
	"8": ffTitle,
	"9": ffDesc,
}

var typeOptions = []string{"all", "issue", "pr"}

// ─── messages ─────────────────────────────────────────────────────────────────

type DashInitMsg struct{}

type dashLoadedMsg struct {
	orgs     []string
	projects []model.Project
}

type dashMetaMsg struct {
	meta      *model.ProjectMeta
	projectID string
}

type dashItemsMsg struct {
	items []model.ProjectItem
}

type dashErrMsg struct{ err error }
type dashStatusSavedMsg struct{ itemID string; statusName string }
type dashIssueSavedMsg struct {
	issue     model.Issue
	newStatus string // empty = unchanged
}

// ─── styles ───────────────────────────────────────────────────────────────────

var (
	listItemSelected = lipgloss.NewStyle().
				Background(lipgloss.Color("#313244")).
				Foreground(lipgloss.Color("#89b4fa")).
				Bold(true)

	listItemNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cdd6f4"))

	detailPane = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#313244")).
			PaddingLeft(1)

	detailHeading = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#cdd6f4"))

	metaKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086"))

	metaVal = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4"))

	openBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("#a6e3a1")).
			Foreground(lipgloss.Color("#1e1e2e")).
			Bold(true).Padding(0, 1)

	closedBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("#f38ba8")).
			Foreground(lipgloss.Color("#1e1e2e")).
			Bold(true).Padding(0, 1)

	statusBadgeStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#313244"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086"))

	editFieldFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#89b4fa")).
				Padding(0, 1)

	editFieldBlurred = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#45475a")).
				Padding(0, 1)

	pickerOverlay = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#89b4fa")).
			Background(lipgloss.Color("#1e1e2e")).
			Padding(0, 1)
)

// ─── Dashboard ────────────────────────────────────────────────────────────────

type Dashboard struct {
	width, height int
	focus         dashFocus

	// filter bar
	activeField filterFieldID

	orgOptions  []string // "me", org1, org2...
	orgIdx      int

	allProjects []model.Project
	projects    []model.Project // filtered by org
	projectIdx  int             // -1 = none selected

	// meta loaded when project selected
	currentProjectID string
	iterations       []model.Iteration // index 0 = "All", 1 = Backlog, then real ones
	iterationIdx     int
	statusOptions    []model.StatusOption // real statuses from project
	statusFieldID    string
	statusIdx        int // 0 = "any"

	typeIdx int // index into typeOptions

	assigneeInput textinput.Model
	labelInput    textinput.Model
	titleInput    textinput.Model
	descInput     textinput.Model

	// issue data
	allItems []model.ProjectItem
	filtered []model.ProjectItem

	// list pane
	listCursor int
	listScroll int
	pendingG   bool

	// detail pane (read)
	viewport  viewport.Model
	vpReady   bool

	// edit mode
	editTitle     textinput.Model
	editBody      textarea.Model
	editStatusIdx int // index into statusOptions (not the "any" prepend)
	editFocus     int // 0=title 1=body 2=status

	// status picker overlay
	pickerCursor int

	loading    bool
	loadingMsg string
	err        string

	client *gh.Client
	user   string
}

func NewDashboard(client *gh.Client, user string) Dashboard {
	d := Dashboard{
		client:     client,
		user:       user,
		projectIdx: -1,
	}

	d.orgOptions = []string{"me"}
	d.iterations = []model.Iteration{{ID: "", Title: "All"}, {ID: gh.BacklogIterationID, Title: "Backlog"}}
	d.statusOptions = nil

	newTI := func(placeholder string, width int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Width = width
		return ti
	}

	d.assigneeInput = newTI("any", 20)
	d.labelInput = newTI("any", 20)
	d.titleInput = newTI("search title...", 28)
	d.descInput = newTI("search description...", 28)

	d.editTitle = newTI("Issue title", 55)
	d.editBody = textarea.New()
	d.editBody.SetWidth(57)
	d.editBody.SetHeight(8)

	d.loading = true
	d.loadingMsg = "Loading projects..."
	return d
}

func (d Dashboard) Init() tea.Cmd {
	return d.loadInitial()
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (d Dashboard) Update(msg tea.Msg) (Dashboard, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = msg.Width, msg.Height
		d.resizeViewport()
		if d.vpReady {
			d.refreshDetailContent()
		}
		return d, nil

	case dashLoadedMsg:
		d.loading = false
		d.orgOptions = append([]string{"me"}, msg.orgs...)
		d.allProjects = msg.projects
		d.filterProjects()
		return d, nil

	case dashMetaMsg:
		if msg.projectID != d.currentProjectID {
			return d, nil // stale response
		}
		d.iterations = []model.Iteration{
			{ID: "", Title: "All"},
			{ID: gh.BacklogIterationID, Title: "Backlog"},
		}
		d.iterations = append(d.iterations, msg.meta.Iterations...)
		d.iterationIdx = 0
		d.statusFieldID = msg.meta.StatusFieldID
		d.statusOptions = msg.meta.StatusOptions
		d.statusIdx = 0
		return d, d.loadItems()

	case dashItemsMsg:
		d.loading = false
		d.allItems = msg.items
		d.applyFilters()
		d.listCursor = 0
		d.listScroll = 0
		if d.vpReady && len(d.filtered) > 0 {
			d.refreshDetailContent()
		}
		return d, nil

	case dashErrMsg:
		d.loading = false
		d.err = msg.err.Error()
		return d, nil

	case dashStatusSavedMsg:
		for i := range d.allItems {
			if d.allItems[i].ID == msg.itemID {
				d.allItems[i].Status = msg.statusName
			}
		}
		d.applyFilters()
		d.focus = focusDetail
		d.refreshDetailContent()
		return d, nil

	case dashIssueSavedMsg:
		for i := range d.allItems {
			if d.allItems[i].Issue.ID == msg.issue.ID {
				d.allItems[i].Issue = msg.issue
				if msg.newStatus != "" {
					d.allItems[i].Status = msg.newStatus
				}
			}
		}
		d.applyFilters()
		d.focus = focusDetail
		d.refreshDetailContent()
		return d, nil

	case tea.KeyMsg:
		return d.handleKey(msg)
	}

	return d.delegateToInputs(msg)
}

// ─── Key handling ─────────────────────────────────────────────────────────────

func (d Dashboard) handleKey(msg tea.KeyMsg) (Dashboard, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+c":
		return d, tea.Quit
	}

	// Quick-jump to filter field: 1–9 work from any focus state except edit/picker
	// (edit mode uses digits in text inputs; picker uses j/k only)
	if d.focus != focusEdit && d.focus != focusStatusPicker {
		if field, ok := filterKey[key]; ok {
			return d.jumpToField(field)
		}
	}

	switch d.focus {
	case focusFilter:
		return d.handleFilterKey(msg, key)
	case focusList:
		return d.handleListKey(key)
	case focusDetail:
		return d.handleDetailKey(key)
	case focusEdit:
		return d.handleEditKey(msg, key)
	case focusStatusPicker:
		return d.handlePickerKey(key)
	}
	return d, nil
}

// jumpToField switches focus to the filter bar, activates the given field, and
// focuses its input if it is a text field.
func (d Dashboard) jumpToField(field filterFieldID) (Dashboard, tea.Cmd) {
	d.blurActiveInput()
	d.focus = focusFilter
	d.activeField = field
	if field >= ffAssignee {
		d.focusActiveInput()
		return d, textinput.Blink
	}
	return d, nil
}

func (d Dashboard) handleFilterKey(rawMsg tea.KeyMsg, key string) (Dashboard, tea.Cmd) {
	isTextField := d.activeField >= ffAssignee

	if isTextField {
		switch key {
		case "tab":
			d.blurActiveInput()
			d.activeField = (d.activeField + 1) % ffCount
			d.focusActiveInput()
			return d, nil
		case "shift+tab":
			d.blurActiveInput()
			d.activeField = (d.activeField - 1 + ffCount) % ffCount
			d.focusActiveInput()
			return d, nil
		case "esc":
			d.blurActiveInput()
			d.focus = focusList
			return d, nil
		case "enter":
			d.blurActiveInput()
			d.applyFilters()
			d.listCursor = 0
			d.listScroll = 0
			d.focus = focusList
			return d, nil
		}
		// delegate to focused text input
		return d.delegateToInputs(rawMsg)
	}

	// dropdown field
	switch key {
	case "tab", "l", "right":
		d.activeField = (d.activeField + 1) % ffCount
		if d.activeField >= ffAssignee {
			d.focusActiveInput()
		}
	case "shift+tab", "h", "left":
		d.activeField = (d.activeField - 1 + ffCount) % ffCount
		if d.activeField >= ffAssignee {
			d.focusActiveInput()
		}
	case "j", "down":
		d.cycleDropdown(1)
		return d.afterDropdownChange()
	case "k", "up":
		d.cycleDropdown(-1)
		return d.afterDropdownChange()
	case "enter":
		d.applyFilters()
		d.listCursor = 0
		d.listScroll = 0
		d.focus = focusList
	case "esc":
		d.focus = focusList
	}
	return d, nil
}

func (d *Dashboard) cycleDropdown(dir int) {
	switch d.activeField {
	case ffOrg:
		d.orgIdx = clamp(d.orgIdx+dir, 0, len(d.orgOptions)-1)
		d.filterProjects()
		d.projectIdx = -1
		d.allItems = nil
		d.filtered = nil
	case ffProject:
		d.projectIdx = clamp(d.projectIdx+dir, -1, len(d.projects)-1)
	case ffIteration:
		d.iterationIdx = clamp(d.iterationIdx+dir, 0, len(d.iterations)-1)
	case ffStatus:
		max := len(d.statusOptions) // 0 = "any"
		d.statusIdx = clamp(d.statusIdx+dir, 0, max)
	case ffType:
		d.typeIdx = clamp(d.typeIdx+dir, 0, len(typeOptions)-1)
	}
}

func (d Dashboard) afterDropdownChange() (Dashboard, tea.Cmd) {
	if d.activeField == ffProject {
		if d.projectIdx >= 0 && d.projectIdx < len(d.projects) {
			p := d.projects[d.projectIdx]
			d.currentProjectID = p.ID
			d.loading = true
			d.loadingMsg = "Loading project..."
			d.allItems = nil
			d.filtered = nil
			return d, d.loadProjectMeta(p.ID)
		}
	}
	d.applyFilters()
	return d, nil
}

func (d Dashboard) handleListKey(key string) (Dashboard, tea.Cmd) {
	n := len(d.filtered)
	vis := d.listVisible()

	switch key {
	case "j", "down":
		if d.listCursor < n-1 {
			d.listCursor++
			d.clampListScroll(vis)
		}
		if d.vpReady {
			d.refreshDetailContent()
		}
	case "k", "up":
		if d.listCursor > 0 {
			d.listCursor--
			d.clampListScroll(vis)
		}
		if d.vpReady {
			d.refreshDetailContent()
		}
	case "g":
		if d.pendingG {
			d.listCursor = 0
			d.listScroll = 0
			d.pendingG = false
			if d.vpReady {
				d.refreshDetailContent()
			}
			return d, nil
		}
		d.pendingG = true
		return d, nil
	case "G":
		d.listCursor = imax(n-1, 0)
		d.listScroll = imax(n-vis, 0)
		if d.vpReady {
			d.refreshDetailContent()
		}
	case "ctrl+d":
		d.listCursor = imin(d.listCursor+vis/2, imax(n-1, 0))
		d.clampListScroll(vis)
		if d.vpReady {
			d.refreshDetailContent()
		}
	case "ctrl+u":
		d.listCursor = imax(d.listCursor-vis/2, 0)
		d.clampListScroll(vis)
		if d.vpReady {
			d.refreshDetailContent()
		}
	case "enter":
		d.focus = focusDetail
		d.refreshDetailContent()
	case "e":
		if n > 0 {
			d.enterEditMode()
		}
	case "o":
		if n > 0 {
			openBrowser(d.filtered[d.listCursor].Issue.URL)
		}
	case "s":
		if n > 0 && len(d.statusOptions) > 0 {
			d.pickerCursor = 0
			d.focus = focusStatusPicker
		}
	case "tab", "f":
		d.focus = focusFilter
		d.activeField = ffOrg
	case "r":
		if d.currentProjectID != "" {
			d.loading = true
			d.loadingMsg = "Refreshing..."
			return d, d.loadItems()
		}
	}
	d.pendingG = false
	return d, nil
}

func (d Dashboard) handleDetailKey(key string) (Dashboard, tea.Cmd) {
	switch key {
	case "j", "down":
		d.viewport.LineDown(1)
	case "k", "up":
		d.viewport.LineUp(1)
	case "ctrl+d":
		d.viewport.HalfViewDown()
	case "ctrl+u":
		d.viewport.HalfViewUp()
	case "g":
		d.viewport.GotoTop()
	case "G":
		d.viewport.GotoBottom()
	case "e":
		d.enterEditMode()
	case "o":
		if len(d.filtered) > 0 {
			openBrowser(d.filtered[d.listCursor].Issue.URL)
		}
	case "s":
		if len(d.statusOptions) > 0 {
			d.pickerCursor = 0
			d.focus = focusStatusPicker
		}
	case "esc", "q":
		d.focus = focusList
	case "tab":
		d.focus = focusFilter
	}
	return d, nil
}

func (d Dashboard) handleEditKey(msg tea.KeyMsg, key string) (Dashboard, tea.Cmd) {
	switch key {
	case "ctrl+s":
		return d, d.saveEdit()
	case "esc":
		d.focus = focusDetail
		return d, nil
	case "tab":
		d.editFocus = (d.editFocus + 1) % 3
		d.syncEditFocus()
	case "shift+tab":
		d.editFocus = (d.editFocus - 1 + 3) % 3
		d.syncEditFocus()
	case "h", "l", "left", "right":
		if d.editFocus == 2 {
			if key == "l" || key == "right" {
				d.editStatusIdx = imin(d.editStatusIdx+1, len(d.statusOptions)-1)
			} else {
				d.editStatusIdx = imax(d.editStatusIdx-1, 0)
			}
			return d, nil
		}
	}

	var cmd tea.Cmd
	switch d.editFocus {
	case 0:
		d.editTitle, cmd = d.editTitle.Update(msg)
	case 1:
		d.editBody, cmd = d.editBody.Update(msg)
	}
	return d, cmd
}

func (d Dashboard) handlePickerKey(key string) (Dashboard, tea.Cmd) {
	switch key {
	case "j", "down":
		if d.pickerCursor < len(d.statusOptions)-1 {
			d.pickerCursor++
		}
	case "k", "up":
		if d.pickerCursor > 0 {
			d.pickerCursor--
		}
	case "enter":
		if len(d.filtered) > 0 && d.pickerCursor < len(d.statusOptions) {
			item := d.filtered[d.listCursor]
			opt := d.statusOptions[d.pickerCursor]
			fieldID := d.statusFieldID
			return d, d.cmdUpdateStatus(item.ID, fieldID, opt.ID, opt.Name)
		}
	case "esc", "q":
		d.focus = focusList
	}
	return d, nil
}

// ─── Input delegation ─────────────────────────────────────────────────────────

func (d Dashboard) delegateToInputs(msg tea.Msg) (Dashboard, tea.Cmd) {
	if d.focus == focusFilter {
		var cmd tea.Cmd
		switch d.activeField {
		case ffAssignee:
			d.assigneeInput, cmd = d.assigneeInput.Update(msg)
		case ffLabel:
			d.labelInput, cmd = d.labelInput.Update(msg)
		case ffTitle:
			d.titleInput, cmd = d.titleInput.Update(msg)
		case ffDesc:
			d.descInput, cmd = d.descInput.Update(msg)
		}
		return d, cmd
	}
	return d, nil
}

func (d *Dashboard) focusActiveInput() {
	d.assigneeInput.Blur()
	d.labelInput.Blur()
	d.titleInput.Blur()
	d.descInput.Blur()
	switch d.activeField {
	case ffAssignee:
		d.assigneeInput.Focus()
	case ffLabel:
		d.labelInput.Focus()
	case ffTitle:
		d.titleInput.Focus()
	case ffDesc:
		d.descInput.Focus()
	}
}

func (d *Dashboard) blurActiveInput() {
	d.assigneeInput.Blur()
	d.labelInput.Blur()
	d.titleInput.Blur()
	d.descInput.Blur()
}

func (d *Dashboard) syncEditFocus() {
	d.editTitle.Blur()
	d.editBody.Blur()
	switch d.editFocus {
	case 0:
		d.editTitle.Focus()
	case 1:
		d.editBody.Focus()
	}
}

// ─── Data helpers ─────────────────────────────────────────────────────────────

func (d *Dashboard) filterProjects() {
	if d.orgIdx == 0 { // "me" = show all
		d.projects = d.allProjects
		return
	}
	org := d.orgOptions[d.orgIdx]
	d.projects = nil
	for _, p := range d.allProjects {
		if p.Owner == org {
			d.projects = append(d.projects, p)
		}
	}
}

func (d *Dashboard) applyFilters() {
	iterFilter := ""
	if d.iterationIdx > 0 && d.iterationIdx < len(d.iterations) {
		iterFilter = d.iterations[d.iterationIdx].ID
	}

	statusFilter := ""
	if d.statusIdx > 0 && d.statusIdx <= len(d.statusOptions) {
		statusFilter = d.statusOptions[d.statusIdx-1].Name
	}

	typeFilter := typeOptions[d.typeIdx]
	assigneeFilter := strings.ToLower(strings.TrimSpace(d.assigneeInput.Value()))
	labelFilter := strings.ToLower(strings.TrimSpace(d.labelInput.Value()))
	titleFilter := strings.ToLower(strings.TrimSpace(d.titleInput.Value()))
	descFilter := strings.ToLower(strings.TrimSpace(d.descInput.Value()))

	d.filtered = nil
	for _, item := range d.allItems {
		if typeFilter == "issue" && item.Issue.State == "" {
			continue
		}

		switch iterFilter {
		case gh.BacklogIterationID:
			if item.IterationID != "" {
				continue
			}
		case "":
			// no filter
		default:
			if item.IterationID != iterFilter {
				continue
			}
		}

		if statusFilter != "" && !strings.EqualFold(item.Status, statusFilter) {
			continue
		}

		if assigneeFilter != "" {
			if !anyContains(item.Issue.Assignees, func(a model.Assignee) string { return a.Login }, assigneeFilter) {
				continue
			}
		}

		if labelFilter != "" {
			if !anyContains(item.Issue.Labels, func(l model.Label) string { return l.Name }, labelFilter) {
				continue
			}
		}

		if titleFilter != "" && !strings.Contains(strings.ToLower(item.Issue.Title), titleFilter) {
			continue
		}

		if descFilter != "" && !strings.Contains(strings.ToLower(item.Issue.Body), descFilter) {
			continue
		}

		d.filtered = append(d.filtered, item)
	}
}

func (d *Dashboard) enterEditMode() {
	if len(d.filtered) == 0 {
		return
	}
	issue := d.filtered[d.listCursor].Issue
	d.editTitle.SetValue(issue.Title)
	d.editBody.SetValue(issue.Body)
	d.editStatusIdx = 0
	for i, s := range d.statusOptions {
		if strings.EqualFold(s.Name, d.filtered[d.listCursor].Status) {
			d.editStatusIdx = i
			break
		}
	}
	d.editFocus = 0
	d.editTitle.Focus()
	d.editBody.Blur()
	d.focus = focusEdit
}

func (d *Dashboard) clampListScroll(vis int) {
	if d.listCursor < d.listScroll {
		d.listScroll = d.listCursor
	}
	if d.listCursor >= d.listScroll+vis {
		d.listScroll = d.listCursor - vis + 1
	}
}

func (d *Dashboard) resizeViewport() {
	_, _, _, contentH := d.layout()
	rightW := d.rightPaneWidth()
	// contentH-1: one line is used by the detail-pane header ("Detail  e edit …")
	vpH := contentH - 1
	if vpH < 1 {
		vpH = 1
	}
	if !d.vpReady {
		d.viewport = viewport.New(rightW-2, vpH)
		d.vpReady = true
	} else {
		d.viewport.Width = rightW - 2
		d.viewport.Height = vpH
	}
}

func (d *Dashboard) refreshDetailContent() {
	if !d.vpReady || len(d.filtered) == 0 {
		return
	}
	d.viewport.SetContent(d.renderDetailContent())
}

func (d Dashboard) listVisible() int {
	_, _, _, contentH := d.layout()
	v := contentH - 1
	if v < 1 {
		return 1
	}
	return v
}

func (d Dashboard) rightPaneWidth() int {
	_, _, leftW, _ := d.layout()
	rw := d.width - leftW - 1
	if rw < 10 {
		return 10
	}
	return rw
}

// layout returns filterH, statusH, leftW, contentH
func (d Dashboard) layout() (filterH, statusH, leftW, contentH int) {
	filterH = 7
	statusH = 1
	leftW = d.width * 38 / 100
	contentH = d.height - filterH - statusH
	if contentH < 1 {
		contentH = 1
	}
	return
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func (d Dashboard) loadInitial() tea.Cmd {
	return func() tea.Msg {
		projects, err := d.client.ListAllProjects(context.Background())
		if err != nil {
			return dashErrMsg{err}
		}
		seen := map[string]bool{}
		var orgs []string
		for _, p := range projects {
			if !seen[p.Owner] {
				seen[p.Owner] = true
				orgs = append(orgs, p.Owner)
			}
		}
		return dashLoadedMsg{orgs: orgs, projects: projects}
	}
}

func (d Dashboard) loadProjectMeta(projectID string) tea.Cmd {
	return func() tea.Msg {
		meta, err := d.client.GetProjectMeta(context.Background(), projectID)
		if err != nil {
			return dashErrMsg{err}
		}
		return dashMetaMsg{meta: meta, projectID: projectID}
	}
}

func (d Dashboard) loadItems() tea.Cmd {
	projectID := d.currentProjectID
	return func() tea.Msg {
		items, err := d.client.GetProjectItems(context.Background(), projectID, "")
		if err != nil {
			return dashErrMsg{err}
		}
		return dashItemsMsg{items}
	}
}

func (d Dashboard) saveEdit() tea.Cmd {
	if len(d.filtered) == 0 {
		return nil
	}
	item := d.filtered[d.listCursor]
	issue := item.Issue

	title := strings.TrimSpace(d.editTitle.Value())
	body := d.editBody.Value()

	if title == "" {
		d.err = "title cannot be empty"
		return nil
	}
	if issue.Owner == "" || issue.Repository == "" {
		d.err = "issue has no associated repository and cannot be updated via the REST API"
		return nil
	}

	// determine whether the project status was changed in the form
	var newStatusOpt *model.StatusOption
	if len(d.statusOptions) > 0 && d.editStatusIdx < len(d.statusOptions) {
		opt := d.statusOptions[d.editStatusIdx]
		if !strings.EqualFold(opt.Name, item.Status) {
			newStatusOpt = &opt
		}
	}

	input := model.UpdateIssueInput{
		Owner:  issue.Owner,
		Repo:   issue.Repository,
		Number: issue.Number,
		Title:  title,
		Body:   body,
		// Labels intentionally omitted (nil) so the API preserves existing labels
	}

	projectID := d.currentProjectID
	itemID := item.ID
	fieldID := d.statusFieldID

	return func() tea.Msg {
		updated, err := d.client.UpdateIssue(context.Background(), input)
		if err != nil {
			return dashErrMsg{err: err}
		}
		newStatusName := ""
		if newStatusOpt != nil {
			if err := d.client.UpdateItemStatus(
				context.Background(), projectID, itemID, fieldID, newStatusOpt.ID,
			); err != nil {
				return dashErrMsg{err: err}
			}
			newStatusName = newStatusOpt.Name
		}
		return dashIssueSavedMsg{issue: *updated, newStatus: newStatusName}
	}
}

func (d Dashboard) cmdUpdateStatus(itemID, fieldID, optionID, optionName string) tea.Cmd {
	projectID := d.currentProjectID
	return func() tea.Msg {
		err := d.client.UpdateItemStatus(context.Background(), projectID, itemID, fieldID, optionID)
		if err != nil {
			return dashErrMsg{err}
		}
		return dashStatusSavedMsg{itemID: itemID, statusName: optionName}
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (d Dashboard) View() string {
	if d.width == 0 {
		return ""
	}
	filterH, _, leftW, contentH := d.layout()
	rightW := d.rightPaneWidth()

	filterBar := d.renderFilterBar(d.width, filterH)

	var left, right string
	if d.focus == focusEdit {
		left = d.renderList(leftW, contentH)
		right = d.renderEditPane(rightW, contentH)
	} else if d.focus == focusStatusPicker {
		left = d.renderList(leftW, contentH)
		right = d.renderDetailWithPicker(rightW, contentH)
	} else {
		left = d.renderList(leftW, contentH)
		right = d.renderDetailPane(rightW, contentH)
	}

	// Build separator without a trailing newline so lipgloss measures it as
	// exactly contentH lines (strings.Repeat("│\n", n) ends with \n → n+1 lines).
	sep := separatorStyle.Render(strings.Repeat("│\n", contentH-1) + "│")
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)

	hint := d.renderHint()
	return lipgloss.JoinVertical(lipgloss.Left, filterBar, content, hint)
}

// ─── Filter bar ───────────────────────────────────────────────────────────────

func (d Dashboard) renderFilterBar(width, _ int) string {
	filterActive := d.focus == focusFilter

	// inputVal returns a display string for a text input.
	inputVal := func(ti textinput.Model, isActive bool) string {
		v := ti.Value()
		if v == "" {
			if isActive {
				return "▌"
			}
			return ti.Placeholder
		}
		if isActive {
			return v + "▌"
		}
		return v
	}

	// distribute splits `total` width among n cells; the last cell absorbs any remainder.
	distribute := func(total, n int) []int {
		if n == 0 {
			return nil
		}
		base := total / n
		ws := make([]int, n)
		for i := range ws {
			ws[i] = base
		}
		ws[n-1] += total - base*n
		return ws
	}

	// makeCell renders a 2-line cell that fills the given outer width.
	// With Padding(0,1), inner content width = cellW-2.  We pre-truncate both
	// lines so lipgloss never wraps them, keeping each cell at exactly 2 lines.
	makeCell := func(field filterFieldID, value string, cellW int) string {
		keyChar := filterLabels[field][:1]
		label := filterLabels[field][2:]
		isActive := filterActive && field == d.activeField

		bgColor := lipgloss.Color("#1e1e2e")
		fgVal := lipgloss.Color("#a6adc8")
		if isActive {
			bgColor = lipgloss.Color("#313244")
			fgVal = lipgloss.Color("#89b4fa")
		}

		// inner is the usable text width inside Padding(0,1)
		inner := cellW - 2
		if inner < 2 {
			inner = 2
		}

		// truncate value to inner width so line2 never wraps
		runes := []rune(value)
		if len(runes) > inner {
			if inner > 1 {
				value = string(runes[:inner-1]) + "…"
			} else {
				value = string(runes[:inner])
			}
		}

		bg := func() lipgloss.Style { return lipgloss.NewStyle().Background(bgColor) }

		keyS := bg().Foreground(lipgloss.Color("#fab387")).Bold(true).Render(keyChar)
		lblS := bg().Foreground(lipgloss.Color("#6c7086")).Render(" " + label + ":")
		valS := bg().Foreground(fgVal).Bold(isActive).Render(value)

		line1 := keyS + lblS
		line2 := valS

		return lipgloss.NewStyle().
			Width(cellW).Height(2).
			Background(bgColor).
			Padding(0, 1).
			Render(line1 + "\n" + line2)
	}

	// ── dropdown values ──────────────────────────────────────────────────────

	orgVal := d.orgOptions[d.orgIdx]

	projVal := "─ select ─"
	if d.projectIdx >= 0 && d.projectIdx < len(d.projects) {
		projVal = d.projects[d.projectIdx].Title
		if len(projVal) > 28 {
			projVal = projVal[:25] + "..."
		}
	}

	iterVal := "All"
	if d.iterationIdx < len(d.iterations) {
		iterVal = d.iterations[d.iterationIdx].Title
	}

	statusVal := "any"
	if d.statusIdx > 0 && d.statusIdx <= len(d.statusOptions) {
		statusVal = d.statusOptions[d.statusIdx-1].Name
	}

	typeVal := typeOptions[d.typeIdx]

	// ── three rows ───────────────────────────────────────────────────────────

	ws1 := distribute(width, 2)
	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		makeCell(ffOrg, orgVal, ws1[0]),
		makeCell(ffProject, projVal, ws1[1]),
	)

	ws2 := distribute(width, 3)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top,
		makeCell(ffIteration, iterVal, ws2[0]),
		makeCell(ffStatus, statusVal, ws2[1]),
		makeCell(ffType, typeVal, ws2[2]),
	)

	ws3 := distribute(width, 4)
	row3 := lipgloss.JoinHorizontal(lipgloss.Top,
		makeCell(ffAssignee, inputVal(d.assigneeInput, filterActive && d.activeField == ffAssignee), ws3[0]),
		makeCell(ffLabel, inputVal(d.labelInput, filterActive && d.activeField == ffLabel), ws3[1]),
		makeCell(ffTitle, inputVal(d.titleInput, filterActive && d.activeField == ffTitle), ws3[2]),
		makeCell(ffDesc, inputVal(d.descInput, filterActive && d.activeField == ffDesc), ws3[3]),
	)

	// Hint line doubles as error display to keep filterH fixed at 7.
	hintText := "  1-9: jump  tab/h/l: next field  j/k: cycle  enter: apply  esc: list"
	hintFg := lipgloss.Color("#6c7086")
	if d.err != "" {
		hintText = "  ✗ " + d.err
		hintFg = lipgloss.Color("#f38ba8")
	}
	hint := lipgloss.NewStyle().
		Background(lipgloss.Color("#1e1e2e")).
		Foreground(hintFg).
		Width(width).
		Render(hintText)

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2, row3, hint)
}

// ─── List pane ────────────────────────────────────────────────────────────────

func (d Dashboard) renderList(w, h int) string {
	header := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("#89b4fa")).Width(w).
		Render(fmt.Sprintf(" Issues (%d)", len(d.filtered)))

	if d.loading {
		body := lipgloss.NewStyle().Width(w).Height(h - 1).
			Foreground(lipgloss.Color("#6c7086")).
			Render("\n  " + d.loadingMsg)
		return lipgloss.JoinVertical(lipgloss.Left, header, body)
	}

	if d.projectIdx < 0 {
		body := lipgloss.NewStyle().Width(w).Height(h-1).
			Foreground(lipgloss.Color("#6c7086")).
			Render("\n  Select a project\n  in the filter bar above")
		return lipgloss.JoinVertical(lipgloss.Left, header, body)
	}

	if len(d.filtered) == 0 {
		body := lipgloss.NewStyle().Width(w).Height(h-1).
			Foreground(lipgloss.Color("#6c7086")).
			Render("\n  No issues match the\n  current filters")
		return lipgloss.JoinVertical(lipgloss.Left, header, body)
	}

	vis := h - 1
	start := d.listScroll
	end := imin(start+vis, len(d.filtered))

	var rows []string
	for i := start; i < end; i++ {
		item := d.filtered[i]
		selected := i == d.listCursor && (d.focus == focusList || d.focus == focusDetail ||
			d.focus == focusEdit || d.focus == focusStatusPicker)

		state := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Render("●")
		if strings.ToLower(item.Issue.State) == "closed" {
			state = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")).Render("●")
		}

		num := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).
			Render(fmt.Sprintf("#%-4d", item.Issue.Number))

		title := item.Issue.Title
		maxTitle := w - 14
		if maxTitle < 5 {
			maxTitle = 5
		}
		if len(title) > maxTitle {
			title = title[:maxTitle-1] + "…"
		}

		row := fmt.Sprintf("%s %s %s", state, num, title)

		if selected {
			rows = append(rows, listItemSelected.Width(w).Render(row))
		} else {
			rows = append(rows, listItemNormal.Width(w).Render(row))
		}
	}

	body := strings.Join(rows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

// ─── Detail pane ──────────────────────────────────────────────────────────────

func (d Dashboard) renderDetailPane(w, h int) string {
	inner := w - 2
	if inner < 4 {
		inner = 4
	}

	if len(d.filtered) == 0 {
		return detailPane.Width(w).Height(h).
			Render(lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).
				Render("Select an issue from the list"))
	}

	// Header must be exactly 1 line and ≤ inner chars wide.
	// Concatenating a pre-padded Width(inner) string with extra text makes
	// the line wider than inner, causing detailPane to wrap it and add a
	// spurious extra line to the pane.  Keybinds are shown in the bottom bar.
	header := lipgloss.NewStyle().Bold(true).Width(inner).
		Foreground(lipgloss.Color("#89b4fa")).Render("Detail")

	var vp string
	if d.vpReady {
		vp = d.viewport.View()
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, vp)
	return detailPane.Width(w).Render(content)
}

func (d Dashboard) renderDetailContent() string {
	if len(d.filtered) == 0 {
		return ""
	}
	item := d.filtered[d.listCursor]
	issue := item.Issue

	var b strings.Builder

	// state badge + number
	badge := openBadge.Render("OPEN")
	if strings.ToLower(issue.State) == "closed" {
		badge = closedBadge.Render("CLOSED")
	}
	b.WriteString(badge + "  " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).
			Render(fmt.Sprintf("%s/%s #%d", issue.Owner, issue.Repository, issue.Number)) + "\n\n")

	// title
	b.WriteString(detailHeading.Render(issue.Title) + "\n\n")

	// metadata
	meta := func(k, v string) string {
		return metaKey.Render(k+": ") + metaVal.Render(v) + "\n"
	}

	if item.Status != "" {
		col := statusColor(item.Status)
		statusRendered := statusBadgeStyle.Background(lipgloss.Color(col)).
			Foreground(lipgloss.Color("#1e1e2e")).Render(item.Status)
		b.WriteString(metaKey.Render("Status: ") + statusRendered + "\n")
	}

	if d.iterationIdx < len(d.iterations) && item.IterationID != "" {
		for _, it := range d.iterations {
			if it.ID == item.IterationID {
				b.WriteString(meta("Sprint", it.Title))
				break
			}
		}
	}

	if len(issue.Assignees) > 0 {
		logins := make([]string, len(issue.Assignees))
		for i, a := range issue.Assignees {
			logins[i] = "@" + a.Login
		}
		b.WriteString(meta("Assignees", strings.Join(logins, ", ")))
	}

	if len(issue.Labels) > 0 {
		var badges []string
		for _, l := range issue.Labels {
			c := "#" + l.Color
			badges = append(badges, lipgloss.NewStyle().
				Background(lipgloss.Color(c)).
				Foreground(lipgloss.Color("#1e1e2e")).
				Padding(0, 1).Render(l.Name))
		}
		b.WriteString(metaKey.Render("Labels: ") + strings.Join(badges, " ") + "\n")
	}

	if !issue.CreatedAt.IsZero() {
		b.WriteString(meta("Created", issue.CreatedAt.Format(time.RFC1123)))
	}
	if !issue.UpdatedAt.IsZero() {
		b.WriteString(meta("Updated", issue.UpdatedAt.Format(time.RFC1123)))
	}

	b.WriteString(metaKey.Render("URL: ") + issue.URL + "\n")

	b.WriteString("\n" + strings.Repeat("─", 40) + "\n\n")

	body := issue.Body
	if body == "" {
		body = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Italic(true).
			Render("No description.")
	}
	b.WriteString(body)
	return b.String()
}

func (d Dashboard) renderDetailWithPicker(w, h int) string {
	if len(d.statusOptions) == 0 {
		return d.renderDetailPane(w, h)
	}

	var lines []string
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89b4fa")).Render(" Quick Status"),
		"",
	)
	for i, opt := range d.statusOptions {
		row := "  " + opt.Name
		if i == d.pickerCursor {
			col := statusColor(opt.Name)
			row = statusBadgeStyle.Background(lipgloss.Color(col)).
				Foreground(lipgloss.Color("#1e1e2e")).Render("▶ " + opt.Name)
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", hintStyle.Render("  j/k: navigate  enter: apply  esc: cancel"))

	content := strings.Join(lines, "\n")
	return detailPane.Width(w).Height(h).Render(content)
}

// ─── Edit pane ────────────────────────────────────────────────────────────────

func (d Dashboard) renderEditPane(w, h int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89b4fa")).
		Render(" Edit Issue") + "\n\n")

	titleBox := editFieldBlurred
	bodyBox := editFieldBlurred
	if d.editFocus == 0 {
		titleBox = editFieldFocused
	}
	if d.editFocus == 1 {
		bodyBox = editFieldFocused
	}

	b.WriteString(metaKey.Render("Title") + "\n")
	b.WriteString(titleBox.Render(d.editTitle.View()) + "\n\n")

	b.WriteString(metaKey.Render("Description") + "\n")
	b.WriteString(bodyBox.Render(d.editBody.View()) + "\n\n")

	if len(d.statusOptions) > 0 {
		b.WriteString(metaKey.Render("Status") + "\n")
		var opts []string
		for i, s := range d.statusOptions {
			if i == d.editStatusIdx {
				col := statusColor(s.Name)
				opts = append(opts, statusBadgeStyle.Background(lipgloss.Color(col)).
					Foreground(lipgloss.Color("#1e1e2e")).Render("◀ "+s.Name+" ▶"))
			} else {
				opts = append(opts, hintStyle.Render("  "+s.Name+"  "))
			}
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, opts...) + "\n\n")
	}

	b.WriteString(hintStyle.Render("tab: next field  h/l: status  ctrl+s: save  esc: cancel"))

	return detailPane.Width(w).Height(h).Render(b.String())
}

// ─── Hint bar ─────────────────────────────────────────────────────────────────

func (d Dashboard) renderHint() string {
	var hint string
	switch d.focus {
	case focusFilter:
		hint = "1-9: jump field  tab/h/l: next  j/k: cycle value  enter: apply  esc: list"
	case focusList:
		hint = "j/k: navigate  enter: detail  e: edit  s: status  o: browser  1-9: filter  r: refresh  ?: help"
	case focusDetail:
		hint = "j/k: scroll  e: edit  s: status  o: browser  1-9: filter  esc: list"
	case focusEdit:
		hint = "tab: next field  h/l: status  ctrl+s: save  esc: cancel"
	case focusStatusPicker:
		hint = "j/k: navigate  enter: apply  esc: cancel"
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1e1e2e")).
		Foreground(lipgloss.Color("#6c7086")).
		Width(d.width).
		Render(" " + hint)
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "start"
	default:
		cmd = "xdg-open"
	}
	exec.Command(cmd, url).Start() //nolint:errcheck
}

func statusColor(status string) string {
	switch strings.ToLower(status) {
	case "todo", "backlog":
		return "#45475a"
	case "in progress":
		return "#fab387"
	case "done", "closed":
		return "#a6e3a1"
	default:
		return "#89b4fa"
	}
}

func anyContains[T any](slice []T, extract func(T) string, substr string) bool {
	for _, v := range slice {
		if strings.Contains(strings.ToLower(extract(v)), substr) {
			return true
		}
	}
	return false
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

