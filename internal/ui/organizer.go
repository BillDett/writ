package ui

import (
	"fmt"
	"os"
	"time"
	"writ/internal/data"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

//////// Organizer

/*

First time Organizer opens- it should re-open the last Document that was open before app quit

CTRL-R - rename currently highlighted item
CTRL-D - duplicate currently highlighted item
//DEL - delete currently highlighted item (after confirmation)
CTRL-F - filter items based on some text (change color to show filtered?)

Also need to be able to switch to Trashed items and restore them individually
(change background color of the Organizer?)
CTRL-T - show trash
CTRL-Z - un-trash currently highlighted item (only when trash is active)

*/

type OrganizerWidget struct {
	*tview.Box
	items     tview.List
	store     data.Store
	window    *MainWindow
	trashmode bool
	item_map  map[int]*data.DocReference // map the List item index to the full DocReference
}

func NewOrganizerWidget(s data.Store) *OrganizerWidget {
	o := &OrganizerWidget{
		Box:      tview.NewBox().SetBorder(true).SetTitle(" writ "),
		items:    *tview.NewList().ShowSecondaryText(false),
		store:    s,
		item_map: make(map[int]*data.DocReference),
	}

	o.SetDrawFunc(o.organizer_draw)

	o.items.SetSelectedBackgroundColor(tview.Styles.ContrastBackgroundColor)

	o.Refresh()

	o.items.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		// Load the text for this non-Trashed Document put into buffer
		if !o.trashmode {
			key := o.item_map[index]
			buffer, err := o.store.LoadDocument(fmt.Sprintf("%d", key.ID))
			if err != nil {
				o.window.Error(err.Error())
			} else {
				if o.window.textwidget.IsModified() {
					err = o.store.SaveDocument(o.window.TextWidget().GetDocKey(),
						o.window.TextWidget().GetText())
					if err != nil {
						o.window.Error(err.Error())
					}
				}
				o.window.TextWidget().SetDocument(fmt.Sprintf("%d", key.ID), mainText, buffer)
			}
		}
	})

	return o
}

// Work backwards from o.item_map to find the index of the given dbkey
func (o *OrganizerWidget) reverseItemMap(dbkey string) int {
	for k, v := range o.item_map {
		if fmt.Sprintf("%d", v.ID) == dbkey {
			return k
		}
	}
	return 0
}

func (o *OrganizerWidget) NewDocument(name string) error {
	if o.window.textwidget.IsModified() {
		err := o.store.SaveDocument(o.window.textwidget.GetDocKey(), o.window.textwidget.GetText())
		if err != nil {
			return err
		}
	}
	id, err := o.store.CreateDocument(name, "")
	if err != nil {
		return err
	}
	o.items.AddItem(name, "", 0, nil)
	o.items.SetCurrentItem(-1)
	docKeyStr := fmt.Sprintf("%d", id)
	o.window.textwidget.SetDocument(docKeyStr, name, "")
	o.item_map[o.items.GetCurrentItem()] = &data.DocReference{ID: int(id), Name: name}
	return nil
}

func (o *OrganizerWidget) Refresh() error {
	refs, err := o.store.ListDocuments(o.trashmode)
	if err != nil {
		return err
	} else {
		clear(o.item_map)
		o.items.Clear()
		for _, v := range refs {
			o.items.AddItem(v.Name, "", 0, nil)
			o.item_map[o.items.GetItemCount()-1] = &v
		}
		return nil
	}
}

func (o *OrganizerWidget) DocumentCount() int { return o.items.GetItemCount() }

func (o *OrganizerWidget) CurrentDocument() (int, string) {
	index := o.items.GetCurrentItem()
	key := o.item_map[index]
	return index, fmt.Sprintf("%d", key.ID)
}

func (o *OrganizerWidget) SetWindow(m *MainWindow) { o.window = m }

func (o *OrganizerWidget) SetTrashmode(t bool) { o.trashmode = t }
func (o *OrganizerWidget) GetTrashmode() bool  { return o.trashmode }

func (o *OrganizerWidget) Focus(delegate func(p tview.Primitive)) {
	o.window.SetLastFocused(o)
	o.Box.Focus(delegate)
}

// TODO: Rethink this a bit- how do we open after the event loop has started?
func (o *OrganizerWidget) OpenLastSeen() error {

	k, err := o.store.LastOpened()
	if err != nil {
		return err
	}

	o.items.SetCurrentItem(o.reverseItemMap(k))

	buffer, err := o.store.LoadDocument(k)
	if err != nil {
		o.window.Error(err.Error())
	} else {
		m, _ := o.items.GetItemText(o.items.GetCurrentItem())
		// NOTE: this obliterates whatever was already in the TextWidget...don't use this with edited text
		o.window.TextWidget().SetDocument(k, m, buffer)
	}
	return nil
}

func (o *OrganizerWidget) Draw(screen tcell.Screen) {
	o.Box.DrawForSubclass(screen, o)
	x, y, width, height := o.GetInnerRect()
	o.items.SetRect(x, y, width, height)
	o.items.Draw(screen)
}

func (o *OrganizerWidget) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return o.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyCtrlT:
			if !o.trashmode {
				o.SetTitle(" writ - Trash ")
				o.items.SetBackgroundColor(tview.Styles.ContrastBackgroundColor)
				o.items.SetSelectedBackgroundColor(tview.Styles.MoreContrastBackgroundColor)
				o.trashmode = true
			} else {
				o.SetTitle(" writ ")
				o.items.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
				o.items.SetSelectedBackgroundColor(tview.Styles.ContrastBackgroundColor)
				o.trashmode = false
			}
			o.Refresh()
		case tcell.KeyCtrlR:
			if !o.trashmode {
				idx := o.items.GetCurrentItem()
				name, _ := o.items.GetItemText(idx)
				msg := fmt.Sprintf("New name for '%s': ", name)
				o.window.CollectInput(msg, o, func(newname string) {
					err := o.store.RenameDocument(fmt.Sprintf("%d", o.item_map[idx].ID), newname)
					if err != nil {
						o.window.Error(err.Error())
					}
					o.Refresh()
				})
			}
		case tcell.KeyCtrlD:
			if !o.trashmode {
				idx := o.items.GetCurrentItem()
				name, _ := o.items.GetItemText(idx)
				msg := fmt.Sprintf("Duplicate '%s' as: ", name)
				o.window.CollectInput(msg, o, func(newname string) {
					_, err := o.store.DuplicateDocument(fmt.Sprintf("%d", o.item_map[idx].ID), newname)
					if err != nil {
						o.window.Error(err.Error())
					}
					o.Refresh()
				})
			}
		case tcell.KeyCtrlP:
			if !o.trashmode {
				idx := o.items.GetCurrentItem()
				name, _ := o.items.GetItemText(idx)
				msg := fmt.Sprintf("Filename to export '%s': ", name)
				o.window.CollectInput(msg, o, func(filename string) {
					err := o.ExportItem(idx, filename)
					if err != nil {
						o.window.Error(err.Error())
					}
				})
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
			name, _ := o.items.GetItemText(o.items.GetCurrentItem())
			if o.trashmode {
				o.window.ShowModal("delselecteddocmodal",
					fmt.Sprintf("Do you want to permanently delete '%s'?", name))

			} else {
				o.window.ShowModal("trashselecteddocmodal",
					fmt.Sprintf("Do you want to move '%s' to Trash?", name))
			}
		case tcell.KeyCtrlZ:
			if o.trashmode {
				key := fmt.Sprintf("%d", o.item_map[o.items.GetCurrentItem()].ID)
				err := o.store.RestoreDocument(key)
				if err != nil {
					o.window.Error(err.Error())
				}
				o.Refresh()
			}

		default:
			if handler := o.items.InputHandler(); handler != nil {
				handler(event, setFocus)
				return
			}
		}
	})
}

func (o *OrganizerWidget) TrashSelectedDocument() {
	key := fmt.Sprintf("%d", o.item_map[o.items.GetCurrentItem()].ID)
	err := o.store.TrashDocument(key)
	if err != nil {
		o.window.Error(err.Error())
	}
}

func (o *OrganizerWidget) DeleteSelectedDocument() {
	key := fmt.Sprintf("%d", o.item_map[o.items.GetCurrentItem()].ID)
	err := o.store.DeleteDocument(key)
	if err != nil {
		o.window.Error(err.Error())
	}
}

func (o *OrganizerWidget) ExportItem(idx int, filename string) error {
	text, err := o.store.LoadDocument(fmt.Sprintf("%d", o.item_map[idx].ID))
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}

// additional draw function for Organizer that further customizes the border
// Adheres to the requirement stated by tview.Box.SetDrawFunc()
func (o *OrganizerWidget) organizer_draw(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	innerx := x + 1
	innery := y + 1
	innerw := width - 2
	innerh := height - 2
	bottom_border := height - 1
	style := tcell.StyleDefault.
		Background(tview.Styles.PrimitiveBackgroundColor).
		Foreground(tview.Styles.PrimaryTextColor)

	// Show updated date for selected document (left-justified)
	if o.items.GetItemCount() > 0 {
		currentIdx := o.items.GetCurrentItem()
		if currentIdx >= 0 && currentIdx < len(o.item_map) {
			if docRef, exists := o.item_map[currentIdx]; exists {
				// Parse the ISO date string and format as MM/DD/YY
				if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", docRef.UpdatedDate); err == nil {
					updatedMsg := fmt.Sprintf(" %s ", parsedTime.Format("01/02/06"))
					for i, r := range updatedMsg {
						screen.SetContent(x+1+i, bottom_border, r, nil, style)
					}
				}
			}
		}
	}

	// Show item count (right-justified)
	tag := "item"
	if o.items.GetItemCount() > 1 {
		tag = "items"
	}
	msg := fmt.Sprintf(" %d %s ", o.items.GetItemCount(), tag)
	startx := x + width - len(msg) - 1 // align right
	for i, r := range msg {
		screen.SetContent(startx+i, bottom_border, r, nil, style)
	}
	return innerx, innery, innerw, innerh
}
