package main

import tui "github.com/gizak/termui"

type TreeItem struct {
	n       *DepsNode
	parent  *TreeItem
	sibling *TreeItem
	child   *TreeItem // pointer to first child
	folded  bool
	total   int // number of (shown) children (not count itself)
}

type ScrollList struct {
	tui.Block // embedded
	Items     *TreeItem
	Curr      *TreeItem

	ItemFgColor  tui.Attribute
	ItemBgColor  tui.Attribute
	FocusFgColor tui.Attribute
	FocusBgColor tui.Attribute

	idx int // current cursor position
	off int // first entry displayed
}

func NewScrollList() *ScrollList {
	l := &ScrollList{Block: *tui.NewBlock()}
	l.ItemFgColor = tui.ThemeAttr("list.item.fg")
	l.ItemBgColor = tui.ThemeAttr("list.item.bg")
	l.FocusFgColor = tui.ColorYellow
	l.FocusBgColor = tui.ColorBlue

	l.idx = 0
	l.off = 0
	return l
}

func (i *TreeItem) next() *TreeItem {
	if i.child == nil || i.folded {
		for i != nil {
			if i.sibling != nil {
				return i.sibling
			}

			i = i.parent
		}
		return nil
	}
	return i.child
}

func (i *TreeItem) expand() {
	if !i.folded || i.child == nil {
		return
	}

	for c := i.child; c != nil; c = c.sibling {
		i.total += c.total + 1
	}

	for p := i.parent; p != nil; p = p.parent {
		p.total += i.total
	}

	i.folded = false
}

func (i *TreeItem) fold() {
	if i.folded || i.child == nil {
		return
	}

	for p := i.parent; p != nil; p = p.parent {
		p.total -= i.total
	}
	i.total = 0

	i.folded = true
}

func (i *TreeItem) toggle() {
	if i.folded {
		i.expand()
	} else {
		i.fold()
	}
}

// Buffer implements Bufferer interface.
func (l *ScrollList) Buffer() tui.Buffer {
	buf := l.Block.Buffer()

	i := 0
	printed := 0

	var ti *TreeItem
	for ti = l.Items; ti != nil; ti = ti.next() {
		if i < l.off {
			i++
			continue
		}
		if printed == l.Height-2 {
			break
		}

		fg := l.ItemFgColor
		bg := l.ItemBgColor
		if i == l.idx {
			fg = l.FocusFgColor
			bg = l.FocusBgColor

			l.Curr = ti
		}

		cs := tui.DefaultTxBuilder.Build(ti.n.name, fg, bg)
		cs = tui.DTrimTxCls(cs, l.Width-2-2-3*ti.n.depth)

		j := 0
		if i == l.idx {
			// draw current line cursor from the beginning
			for j < 3*ti.n.depth {
				buf.Set(j+1, printed+1, tui.Cell{' ', fg, bg})
				j++
			}
		} else {
			j = 3 * ti.n.depth
		}

		if ti.folded {
			buf.Set(j+1, printed+1, tui.Cell{'+', fg, bg})
		} else {
			buf.Set(j+1, printed+1, tui.Cell{'-', fg, bg})
		}
		buf.Set(j+2, printed+1, tui.Cell{' ', fg, bg})
		j += 2

		for _, vv := range cs {
			w := vv.Width()
			buf.Set(j+1, printed+1, vv)
			j += w
		}

		printed++
		i++

		if i != l.idx+1 {
			continue
		}

		// draw current line cursor to the end
		for j < l.Width-2 {
			buf.Set(j+1, printed, tui.Cell{' ', fg, bg})
			j++
		}
	}
	return buf
}

func (l *ScrollList) Down() {
	if l.idx < l.Items.total {
		l.idx++
	}
	if l.idx-l.off >= l.Height-2 {
		l.off++
	}
}

func (l *ScrollList) Up() {
	if l.idx > 0 {
		l.idx--
	}
	if l.idx < l.off {
		l.off = l.idx
	}
}

func (l *ScrollList) Toggle() {
	l.Curr.toggle()
}

func makeItems(dep *DepsNode, parent *TreeItem) *TreeItem {
	item := &TreeItem{n: dep, parent: parent, folded: false, total: len(dep.child)}

	var prev *TreeItem
	for _, v := range dep.child {
		c := makeItems(v, item)

		if item.child == nil {
			item.child = c
		}
		if prev != nil {
			prev.sibling = c
		}
		prev = c

		item.total += c.total
	}
	return item
}

func ShowWithTUI(dep *DepsNode) {
	if err := tui.Init(); err != nil {
		panic(err)
	}
	defer tui.Close()

	items := makeItems(dep, nil)

	ls := NewScrollList()

	ls.BorderLabel = "Tree view"
	ls.Height = tui.TermHeight()
	ls.Width = tui.TermWidth()
	ls.Items = items
	ls.Curr = items

	tui.Render(ls)

	// handle key pressing
	tui.Handle("/sys/kbd/q", func(tui.Event) {
		// press q to quit
		tui.StopLoop()
	})
	tui.Handle("/sys/kbd/C-c", func(tui.Event) {
		// press Ctrl-C to quit
		tui.StopLoop()
	})

	tui.Handle("/sys/kbd/<down>", func(tui.Event) {
		ls.Down()
		tui.Render(ls)
	})
	tui.Handle("/sys/kbd/<up>", func(tui.Event) {
		ls.Up()
		tui.Render(ls)
	})

	tui.Handle("/sys/kbd/<enter>", func(tui.Event) {
		ls.Toggle()
		tui.Render(ls)
	})

	tui.Handle("/sys/wnd/resize", func(tui.Event) {
		ls.Height = tui.TermHeight()
		ls.Width = tui.TermWidth()
		tui.Render(ls)
	})

	tui.Loop()
}
