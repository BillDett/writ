package ui

import (
	_ "embed"
	"fmt"
	"time"
	"writ/internal/data"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

//go:embed resources/help.txt
var helptext string

// Override the global Styles fields for the colors we want
func setStyles() {
	/*
		tview.Styles.PrimitiveBackgroundColor = tcell.ColorMediumBlue
		tview.Styles.ContrastBackgroundColor = tcell.ColorTeal
		tview.Styles.MoreContrastBackgroundColor = tcell.ColorYellow
	*/

	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorTeal
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorYellow
	tview.Styles.BorderColor = tcell.ColorWhite
	tview.Styles.TitleColor = tcell.ColorYellow
	tview.Styles.GraphicsColor = tcell.ColorWhite
	tview.Styles.PrimaryTextColor = tcell.ColorWhite
	tview.Styles.SecondaryTextColor = tcell.ColorGreen
	tview.Styles.TertiaryTextColor = tcell.ColorYellow
	tview.Styles.InverseTextColor = tcell.ColorBlue
	tview.Styles.ContrastSecondaryTextColor = tcell.ColorNavy

}

type MainWindow struct {
	*tview.Application
	mainView        *tview.Grid
	modal_open      bool
	organizer_focus bool
	last_focused    tview.Primitive
	pages           *tview.Pages
	textwidget      *TextWidget
	organizerwidget *OrganizerWidget
	inputField      *tview.InputField
	modals          map[string]*tview.Modal
	store           data.Store
}

func (m *MainWindow) createModals() {

	m.modals["quitmodal"] = tview.NewModal().
		SetText("Do you want to quit the application?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				m.Stop()
			} else {
				m.closeModal()
			}
		})

	m.modals["errormodal"] = tview.NewModal().
		SetText("Error!").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			m.closeModal()
		})

	m.modals["trashselecteddocmodal"] = tview.NewModal().
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				m.OrganizerWidget().TrashSelectedDocument()
				_, key := m.organizerwidget.CurrentDocument()
				// Did we just trash the currently opened Document?
				if m.textwidget.GetDocKey() == key {
					// Need to clear out the textwidget
					// TODO: NEED A PROPER 'empty' STATE FOR TextWidget
					m.textwidget.SetText("")
				}
				m.OrganizerWidget().Refresh()
				m.closeModal()
			} else {
				m.closeModal()
			}
		})

	m.modals["delselecteddocmodal"] = tview.NewModal().
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				m.OrganizerWidget().DeleteSelectedDocument()
				m.OrganizerWidget().Refresh()
				m.closeModal()
			} else {
				m.closeModal()
			}
		})
}

func NewMainWindow(s data.Store) *MainWindow {

	setStyles()

	m := &MainWindow{
		Application:     tview.NewApplication(),
		organizer_focus: false,
		modal_open:      false,
		pages:           tview.NewPages(),
		textwidget:      NewTextWidget(),
		organizerwidget: NewOrganizerWidget(s),
		inputField:      tview.NewInputField(),
		store:           s,
	}

	m.modals = make(map[string]*tview.Modal)
	m.createModals()

	m.organizerwidget.SetWindow(m)
	m.organizerwidget.SetTitleAlign(tview.AlignLeft)

	m.textwidget.SetWindow(m).
		SetStyle(tcell.StyleDefault.
			Background(tview.Styles.PrimitiveBackgroundColor).
			Foreground(tview.Styles.PrimaryTextColor))
	m.textwidget.SetTitleAlign(tview.AlignLeft)

	m.mainView = tview.NewGrid().
		SetRows(0, 1).
		AddItem(m.organizerwidget, 0, 0, 1, 1, 0, 0, false).
		AddItem(m.textwidget, 0, 1, 1, 4, 0, 0, false)

	m.pages.AddPage("mainview", m.mainView, true, true)

	helpflex := tview.NewFlex().SetDirection(tview.FlexRow)
	helpbox := tview.NewTextView().
		SetTextColor(tview.Styles.PrimaryTextColor).
		SetWrap(true)
	fmt.Fprint(helpbox, helptext)
	helpflex.AddItem(helpbox, 0, 1, false)

	m.pages.AddPage("help", helpflex, true, false)

	m.SetInputCapture(m.HandleEvent)

	// Start the background saver with this delay
	// TODO: Should pick up this delay from config
	go m.backgroundSaver(20)

	m.SetRoot(m.pages, true).EnableMouse(true).EnablePaste(true).SetFocus((m.organizerwidget))

	// If we have some documents, let's open the last seen one
	if m.organizerwidget.DocumentCount() > 0 {
		err := m.organizerwidget.OpenLastSeen()
		if err != nil {
			m.Error(err.Error())
		}
	}

	return m
}

func (m *MainWindow) Init() *MainWindow {
	// If we have a new, empty database, prompt for a new document
	m.promptIfNew()
	return m
}

// PostStartup performs actions that need to happen after the event loop starts
func (m *MainWindow) PostStartup() {
	if m.organizerwidget.DocumentCount() > 0 {
		err := m.organizerwidget.OpenLastSeen()
		if err != nil {
			m.Error(err.Error())
		}
	}
}

func (m *MainWindow) HandleEvent(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC: // override default tview where CTRL-C quits app
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	case tcell.KeyCtrlQ:
		//m.ShowModal("quitmodal", "")
		// TODO: Force a save on current document
		m.Stop()
	case tcell.KeyESC:
		name, _ := m.pages.GetFrontPage()
		if name == "modal" {
			// Pass along if a modal is open (it should close the modal)
			return event
		} else if name == "help" {
			m.pages.SwitchToPage("mainview")
			m.SetFocus(m.last_focused)
		}
	case tcell.KeyCtrlO:
		if m.textwidget.IsModified() {
			m.store.SaveDocument(m.textwidget.GetDocKey(), m.textwidget.GetText())
		}
		// Refresh the organizer to pick up any updated dates from recent saves
		err := m.organizerwidget.Refresh()
		if err != nil {
			m.Error(err.Error())
		}
		m.SetFocus(m.organizerwidget)
	case tcell.KeyCtrlE:
		if !m.promptIfNew() { // don't go into editing if we don't have a document yet or are showing Trash
			if !m.organizerwidget.GetTrashmode() {
				m.SetFocus(m.textwidget)
			}
		}
	case tcell.KeyCtrlN:
		m.CollectInput("New document name: ", m.textwidget, func(name string) {
			err := m.organizerwidget.NewDocument(name)
			if err != nil {
				m.Error(err.Error())
			}
		})
	case tcell.KeyF1:
		m.pages.ShowPage("help")
		//m.SetFocus(helpbox)
	}

	return event
}

func (m *MainWindow) Error(text string) { m.ShowModal("errormodal", text) }

func (m *MainWindow) TextWidget() *TextWidget           { return m.textwidget }
func (m *MainWindow) OrganizerWidget() *OrganizerWidget { return m.organizerwidget }

func (m *MainWindow) ShowModal(name string, text string) {
	modal := m.modals[name]
	if modal != nil {
		if text != "" {
			modal.SetText(text)
		}
		m.pages.AddPage("modal", modal, false, true)
		m.pages.ShowPage("modal")
		m.EnableMouse(false)
	}
}

func (m *MainWindow) SetLastFocused(p tview.Primitive) { m.last_focused = p }
func (m *MainWindow) GetLastFocused() tview.Primitive  { return m.last_focused }

// CollectInput prompts the user for an input string and passes is to 'handler', and then passing focus to 'delegate'.
// If the user abandons the input with ESC/TAB, the focus goes back to the last focused primitive.
func (m *MainWindow) CollectInput(label string, delegate tview.Primitive, handler func(response string)) {
	m.inputField.SetLabel(label).
		SetText("").
		SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyESC, tcell.KeyTAB:
				m.pages.ShowPage("mainview")
				m.SetFocus(m.last_focused)
			case tcell.KeyEnter:
				input := m.inputField.GetText()
				if input != "" {
					handler(input)
					m.SetFocus(delegate)
				} else {
					m.SetFocus(m.last_focused)
				}
			}
			m.mainView.RemoveItem(m.inputField)
		})
	m.mainView.AddItem(m.inputField, 1, 0, 1, 5, 0, 0, false)
	m.SetFocus(m.inputField)
}

func (m *MainWindow) closeModal() {
	m.pages.RemovePage("modal")
	m.EnableMouse(true)
	m.SetFocus(m.last_focused)
}

func (m *MainWindow) promptIfNew() bool {
	empty := m.OrganizerWidget().DocumentCount() == 0
	if empty {
		m.Error("Use CTRL-N to create a new Document")
	}
	return empty
}

// backgroundSaver should be invoked as a goroutine- it wakes up every 'delay' seconds to save the text in the editor if it's dirty.
func (m *MainWindow) backgroundSaver(delay int) {
	ticker := time.NewTicker(time.Duration(delay) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if m.textwidget.dirty {
			m.textwidget.window.store.SaveDocument(m.textwidget.currentDocKey, m.textwidget.GetText())
			m.textwidget.dirty = false
			m.Draw()
		}
	}
}
