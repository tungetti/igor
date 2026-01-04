package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tungetti/igor/internal/ui/theme"
)

// ListItem represents an item in the list.
// It implements the list.Item interface from bubbles.
type ListItem struct {
	title       string
	description string
	value       interface{}
}

// Title returns the item's title for display.
func (i ListItem) Title() string { return i.title }

// Description returns the item's description for display.
func (i ListItem) Description() string { return i.description }

// FilterValue returns the value used for filtering.
func (i ListItem) FilterValue() string { return i.title }

// Value returns the item's associated value.
func (i ListItem) Value() interface{} { return i.value }

// NewListItem creates a new list item with title, description, and value.
func NewListItem(title, description string, value interface{}) ListItem {
	return ListItem{title: title, description: description, value: value}
}

// NewSimpleListItem creates a new list item with just a title.
func NewSimpleListItem(title string, value interface{}) ListItem {
	return ListItem{title: title, value: value}
}

// ListModel wraps bubbles list with Igor styling.
// It provides a scrollable list with item selection.
type ListModel struct {
	list   list.Model
	styles theme.Styles
	width  int
	height int
}

// NewList creates a new list component with the specified items.
// The list uses Igor styling and disables filtering by default.
func NewList(styles theme.Styles, title string, items []ListItem, width, height int) ListModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	l := list.New(listItems, list.NewDefaultDelegate(), width, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	// Apply Igor styles
	l.Styles.Title = styles.Title
	l.Styles.TitleBar = styles.Header

	return ListModel{
		list:   l,
		styles: styles,
		width:  width,
		height: height,
	}
}

// NewListWithDelegate creates a new list with a custom delegate.
func NewListWithDelegate(styles theme.Styles, title string, items []ListItem, delegate list.ItemDelegate, width, height int) ListModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	l := list.New(listItems, delegate, width, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	// Apply Igor styles
	l.Styles.Title = styles.Title
	l.Styles.TitleBar = styles.Header

	return ListModel{
		list:   l,
		styles: styles,
		width:  width,
		height: height,
	}
}

// Init implements tea.Model. List doesn't need initialization.
func (m ListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and handles list navigation.
func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model and renders the list.
func (m ListModel) View() string {
	return m.list.View()
}

// SelectedItem returns the currently selected item and whether one is selected.
func (m ListModel) SelectedItem() (ListItem, bool) {
	item := m.list.SelectedItem()
	if item == nil {
		return ListItem{}, false
	}
	listItem, ok := item.(ListItem)
	if !ok {
		return ListItem{}, false
	}
	return listItem, true
}

// SelectedIndex returns the index of the currently selected item.
func (m ListModel) SelectedIndex() int {
	return m.list.Index()
}

// SetItems replaces all items in the list.
func (m *ListModel) SetItems(items []ListItem) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	m.list.SetItems(listItems)
}

// Items returns all items in the list.
func (m ListModel) Items() []ListItem {
	items := m.list.Items()
	result := make([]ListItem, 0, len(items))
	for _, item := range items {
		if listItem, ok := item.(ListItem); ok {
			result = append(result, listItem)
		}
	}
	return result
}

// SetSize updates the list's dimensions.
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
}

// Width returns the list's width.
func (m ListModel) Width() int {
	return m.width
}

// Height returns the list's height.
func (m ListModel) Height() int {
	return m.height
}

// SetTitle updates the list's title.
func (m *ListModel) SetTitle(title string) {
	m.list.Title = title
}

// Title returns the list's title.
func (m ListModel) Title() string {
	return m.list.Title
}

// Len returns the number of items in the list.
func (m ListModel) Len() int {
	return len(m.list.Items())
}

// IsEmpty returns true if the list has no items.
func (m ListModel) IsEmpty() bool {
	return m.Len() == 0
}

// Select moves the selection to the specified index.
func (m *ListModel) Select(index int) {
	m.list.Select(index)
}

// EnableFiltering enables the list's filtering functionality.
func (m *ListModel) EnableFiltering() {
	m.list.SetFilteringEnabled(true)
}

// DisableFiltering disables the list's filtering functionality.
func (m *ListModel) DisableFiltering() {
	m.list.SetFilteringEnabled(false)
}

// ShowStatusBar shows the list's status bar.
func (m *ListModel) ShowStatusBar() {
	m.list.SetShowStatusBar(true)
}

// HideStatusBar hides the list's status bar.
func (m *ListModel) HideStatusBar() {
	m.list.SetShowStatusBar(false)
}

// ShowHelp shows the list's help.
func (m *ListModel) ShowHelp() {
	m.list.SetShowHelp(true)
}

// HideHelp hides the list's help.
func (m *ListModel) HideHelp() {
	m.list.SetShowHelp(false)
}
