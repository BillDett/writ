package ui

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"writ/internal/data"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ItemMap interface for mapping list indices to database IDs bidirectionally
type ItemMap interface {
	// CRUD operations
	Set(listIndex int, docRef *data.DocReference)
	Get(listIndex int) (*data.DocReference, bool)
	GetDBKey(listIndex int) (string, bool) // returns string version of DB ID
	GetListIndex(dbKey string) (int, bool) // reverse lookup
	Delete(listIndex int)
	Clear()
	Count() int
	HasIndex(listIndex int) bool
}

// itemMapImpl implements ItemMap
type itemMapImpl struct {
	indexToRef map[int]*data.DocReference
}

// NewItemMap creates a new ItemMap instance
func NewItemMap() ItemMap {
	return &itemMapImpl{
		indexToRef: make(map[int]*data.DocReference),
	}
}

func (im *itemMapImpl) Set(listIndex int, docRef *data.DocReference) {
	im.indexToRef[listIndex] = docRef
}

func (im *itemMapImpl) Get(listIndex int) (*data.DocReference, bool) {
	docRef, exists := im.indexToRef[listIndex]
	return docRef, exists
}

func (im *itemMapImpl) GetDBKey(listIndex int) (string, bool) {
	if docRef, exists := im.indexToRef[listIndex]; exists {
		return strconv.Itoa(docRef.ID), true
	}
	return "", false
}

func (im *itemMapImpl) GetListIndex(dbKey string) (int, bool) {
	for index, docRef := range im.indexToRef {
		if strconv.Itoa(docRef.ID) == dbKey {
			return index, true
		}
	}
	return 0, false
}

func (im *itemMapImpl) Delete(listIndex int) {
	delete(im.indexToRef, listIndex)
}

func (im *itemMapImpl) Clear() {
	clear(im.indexToRef)
}

func (im *itemMapImpl) Count() int {
	return len(im.indexToRef)
}

func (im *itemMapImpl) HasIndex(listIndex int) bool {
	_, exists := im.indexToRef[listIndex]
	return exists
}

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
	itemMap   ItemMap // bidirectional mapping between list index and database ID
}

func NewOrganizerWidget(s data.Store) *OrganizerWidget {
	o := &OrganizerWidget{
		Box:     tview.NewBox().SetBorder(true).SetTitle(" writ "),
		items:   *tview.NewList().ShowSecondaryText(false),
		store:   s,
		itemMap: NewItemMap(),
	}

	o.SetDrawFunc(o.organizer_draw)

	o.items.SetSelectedBackgroundColor(tview.Styles.ContrastBackgroundColor)

	o.Refresh()

	o.items.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		// Load the text for this non-Trashed Document put into buffer
		if !o.trashmode {
			if dbKey, ok := o.itemMap.GetDBKey(index); ok {
				buffer, err := o.store.LoadDocument(dbKey)
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
					o.window.TextWidget().SetDocument(dbKey, mainText, buffer)
				}
			}
		}
	})

	return o
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
	docKeyStr := strconv.FormatInt(id, 10)
	o.window.textwidget.SetDocument(docKeyStr, name, "")
	o.itemMap.Set(o.items.GetCurrentItem(), &data.DocReference{ID: int(id), Name: name})
	return nil
}

func (o *OrganizerWidget) Refresh() error {
	refs, err := o.store.ListDocuments(o.trashmode, data.SortByUpdatedDate)
	if err != nil {
		return err
	} else {
		o.itemMap.Clear()
		o.items.Clear()
		for _, v := range refs {
			o.items.AddItem(v.Name, "", 0, nil)
			o.itemMap.Set(o.items.GetItemCount()-1, &v)
		}
		return nil
	}
}

func (o *OrganizerWidget) DocumentCount() int { return o.items.GetItemCount() }

func (o *OrganizerWidget) CurrentDocument() (int, string) {
	index := o.items.GetCurrentItem()
	dbKey, _ := o.itemMap.GetDBKey(index)
	return index, dbKey
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

	if listIndex, ok := o.itemMap.GetListIndex(k); ok {
		o.items.SetCurrentItem(listIndex)
	}

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
					if dbKey, ok := o.itemMap.GetDBKey(idx); ok {
						err := o.store.RenameDocument(dbKey, newname)
						if err != nil {
							o.window.Error(err.Error())
						}
						o.Refresh()
					}
				})
			}
		case tcell.KeyCtrlD:
			if !o.trashmode {
				idx := o.items.GetCurrentItem()
				name, _ := o.items.GetItemText(idx)
				msg := fmt.Sprintf("Duplicate '%s' as: ", name)
				o.window.CollectInput(msg, o, func(newname string) {
					if dbKey, ok := o.itemMap.GetDBKey(idx); ok {
						_, err := o.store.DuplicateDocument(dbKey, newname)
						if err != nil {
							o.window.Error(err.Error())
						}
						o.Refresh()
					}
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
				if dbKey, ok := o.itemMap.GetDBKey(o.items.GetCurrentItem()); ok {
					err := o.store.RestoreDocument(dbKey)
					if err != nil {
						o.window.Error(err.Error())
					}
					o.Refresh()
				}
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
	if dbKey, ok := o.itemMap.GetDBKey(o.items.GetCurrentItem()); ok {
		err := o.store.TrashDocument(dbKey)
		if err != nil {
			o.window.Error(err.Error())
		}
	}
}

func (o *OrganizerWidget) DeleteSelectedDocument() {
	if dbKey, ok := o.itemMap.GetDBKey(o.items.GetCurrentItem()); ok {
		err := o.store.DeleteDocument(dbKey)
		if err != nil {
			o.window.Error(err.Error())
		}
	}
}

func (o *OrganizerWidget) ExportItem(idx int, filename string) error {
	if dbKey, ok := o.itemMap.GetDBKey(idx); ok {
		text, err := o.store.LoadDocument(dbKey)
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
		if currentIdx >= 0 && o.itemMap.HasIndex(currentIdx) {
			if docRef, exists := o.itemMap.Get(currentIdx); exists {
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
