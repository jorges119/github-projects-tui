package keymap

import "github.com/charmbracelet/bubbles/key"

type ListKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding
	Select   key.Binding
	Back     key.Binding
	New      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Search   key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

type FormKeyMap struct {
	NextField key.Binding
	PrevField key.Binding
	Submit    key.Binding
	Cancel    key.Binding
}

var List = ListKeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("gg", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	),
	HalfUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "half up"),
	),
	HalfDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "half down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("q", "esc", "backspace"),
		key.WithHelp("q/esc", "back"),
	),
	New: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

var Form = FormKeyMap{
	NextField: key.NewBinding(
		key.WithKeys("tab", "down"),
		key.WithHelp("tab", "next field"),
	),
	PrevField: key.NewBinding(
		key.WithKeys("shift+tab", "up"),
		key.WithHelp("shift+tab", "prev field"),
	),
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func (k ListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.New, k.Edit, k.Search, k.Help, k.Quit}
}

func (k ListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.HalfUp, k.HalfDown},
		{k.Select, k.Back, k.New, k.Edit, k.Delete},
		{k.Search, k.Refresh, k.Help, k.Quit},
	}
}
