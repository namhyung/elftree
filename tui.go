package main

import tui "github.com/gizak/termui"

type TreeItem struct {
	node    *DepsNode
	parent  *TreeItem
	sibling *TreeItem
	child   *TreeItem // pointer to first child
	folded  bool
	total   int // number of (shown) children (not count itself)
}

type TreeView struct {
	tui.Block // embedded
	Root      *TreeItem
	Curr      *TreeItem

	ItemFgColor  tui.Attribute
	ItemBgColor  tui.Attribute
	FocusFgColor tui.Attribute
	FocusBgColor tui.Attribute

	idx int // current cursor position
	off int // first entry displayed

	rows int
	cols int
}

func NewTreeView() *TreeView {
	tv := &TreeView{Block: *tui.NewBlock()}

	tv.ItemFgColor = tui.ThemeAttr("list.item.fg")
	tv.ItemBgColor = tui.ThemeAttr("list.item.bg")
	tv.FocusFgColor = tui.ColorYellow
	tv.FocusBgColor = tui.ColorBlue

	tv.idx = 0
	tv.off = 0
	return tv
}

func (ti *TreeItem) next() *TreeItem {
	if ti.child == nil || ti.folded {
		for ti != nil {
			if ti.sibling != nil {
				return ti.sibling
			}

			ti = ti.parent
		}
		return nil
	}
	return ti.child
}

func (ti *TreeItem) expand() {
	if !ti.folded || ti.child == nil {
		return
	}

	for c := ti.child; c != nil; c = c.sibling {
		ti.total += c.total + 1
	}

	for p := ti.parent; p != nil; p = p.parent {
		p.total += ti.total
	}

	ti.folded = false
}

func (ti *TreeItem) fold() {
	if ti.folded || ti.child == nil {
		return
	}

	for p := ti.parent; p != nil; p = p.parent {
		p.total -= ti.total
	}
	ti.total = 0

	ti.folded = true
}

func (ti *TreeItem) toggle() {
	if ti.folded {
		ti.expand()
	} else {
		ti.fold()
	}
}

// Buffer implements Bufferer interface.
func (tv *TreeView) Buffer() tui.Buffer {
	buf := tv.Block.Buffer()

	i := 0
	printed := 0

	var ti *TreeItem
	for ti = tv.Root; ti != nil; ti = ti.next() {
		if i < tv.off {
			i++
			continue
		}
		if printed == tv.rows {
			break
		}

		fg := tv.ItemFgColor
		bg := tv.ItemBgColor
		if i == tv.idx {
			fg = tv.FocusFgColor
			bg = tv.FocusBgColor

			tv.Curr = ti
		}

		indent := 3 * ti.node.depth
		cs := tui.DefaultTxBuilder.Build(ti.node.name, fg, bg)
		cs = tui.DTrimTxCls(cs, tv.cols-2-indent)

		j := 0
		if i == tv.idx {
			// draw current line cursor from the beginning
			for j < indent {
				buf.Set(j+1, printed+1, tui.Cell{' ', fg, bg})
				j++
			}
		} else {
			j = indent
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

		if i != tv.idx+1 {
			continue
		}

		// draw current line cursor to the end
		for j < tv.cols {
			buf.Set(j+1, printed, tui.Cell{' ', fg, bg})
			j++
		}
	}
	return buf
}

func (tv *TreeView) Down() {
	if tv.idx < tv.Root.total {
		tv.idx++
	}
	if tv.idx-tv.off >= tv.rows {
		tv.off++
	}
}

func (tv *TreeView) Up() {
	if tv.idx > 0 {
		tv.idx--
	}
	if tv.idx < tv.off {
		tv.off = tv.idx
	}
}

func (tv *TreeView) PageDown() {
	bottom := tv.off + tv.rows - 1
	if bottom > tv.Root.total {
		bottom = tv.Root.total
	}

	// At first, move to the bottom of current page
	if tv.idx != bottom {
		tv.idx = bottom
		return
	}

	tv.idx += tv.rows
	if tv.idx > tv.Root.total {
		tv.idx = tv.Root.total
	}
	if tv.idx-tv.off >= tv.rows {
		tv.off = tv.idx - tv.rows + 1
	}
}

func (tv *TreeView) PageUp() {
	// At first, move to the top of current page
	if tv.idx != tv.off {
		tv.idx = tv.off
		return
	}

	tv.idx -= tv.rows
	if tv.idx < 0 {
		tv.idx = 0
	}

	tv.off = tv.idx
}

func (tv *TreeView) Home() {
	tv.idx = 0
	tv.off = 0
}

func (tv *TreeView) End() {
	tv.idx = tv.Root.total
	tv.off = tv.idx - tv.rows + 1

	if tv.off < 0 {
		tv.off = 0
	}
}

func (tv *TreeView) Toggle() {
	tv.Curr.toggle()
}

func makeItems(dep *DepsNode, parent *TreeItem) *TreeItem {
	item := &TreeItem{node: dep, parent: parent, folded: false, total: len(dep.child)}

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

	root := makeItems(dep, nil)

	tv := NewTreeView()

	tv.BorderLabel = "ELF Tree"
	tv.Height = tui.TermHeight()
	tv.Width = tui.TermWidth()
	tv.Root = root
	tv.Curr = root

	tv.rows = tv.Height - 2 // exclude border at top and bottom
	tv.cols = tv.Width - 2  // exclude border at left and right

	tui.Render(tv)

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
		tv.Down()
		tui.Render(tv)
	})
	tui.Handle("/sys/kbd/<up>", func(tui.Event) {
		tv.Up()
		tui.Render(tv)
	})
	tui.Handle("/sys/kbd/<next>", func(tui.Event) {
		tv.PageDown()
		tui.Render(tv)
	})
	tui.Handle("/sys/kbd/<previous>", func(tui.Event) {
		tv.PageUp()
		tui.Render(tv)
	})
	tui.Handle("/sys/kbd/<home>", func(tui.Event) {
		tv.Home()
		tui.Render(tv)
	})
	tui.Handle("/sys/kbd/<end>", func(tui.Event) {
		tv.End()
		tui.Render(tv)
	})

	tui.Handle("/sys/kbd/<enter>", func(tui.Event) {
		tv.Toggle()
		tui.Render(tv)
	})

	tui.Handle("/sys/wnd/resize", func(tui.Event) {
		tv.Height = tui.TermHeight()
		tv.Width = tui.TermWidth()
		tv.rows = tv.Height - 2
		tv.cols = tv.Width - 2
		tui.Render(tv)
	})

	tui.Loop()
}
