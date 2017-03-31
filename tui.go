package main

import tui "github.com/gizak/termui"

type TreeItem struct {
	node   interface{}
	parent *TreeItem
	prev   *TreeItem // pointer to siblings
	next   *TreeItem
	child  *TreeItem // pointer to first child
	folded bool
	total  int // number of (shown) children (not count itself)
}

type TreeView struct {
	tui.Block // embedded
	Root      *TreeItem
	Top       *TreeItem
	Curr      *TreeItem

	ItemFgColor  tui.Attribute
	ItemBgColor  tui.Attribute
	FocusFgColor tui.Attribute
	FocusBgColor tui.Attribute

	idx int // current cursor position
	off int // first entry displayed
	pos int // position of x-axis

	rows int
	cols int
}

type StatusLine struct {
	tui.Block // embedded
	tv        *TreeView
}

func NewTreeView() *TreeView {
	tv := &TreeView{Block: *tui.NewBlock()}

	tv.ItemFgColor = tui.ThemeAttr("list.item.fg")
	tv.ItemBgColor = tui.ThemeAttr("list.item.bg")
	tv.FocusFgColor = tui.ColorYellow
	tv.FocusBgColor = tui.ColorBlue

	tv.BorderLabel = "ELF Tree"

	tv.idx = 0
	tv.off = 0
	return tv
}

func NewStatusLine(tv *TreeView) *StatusLine {
	sl := &StatusLine{Block: *tui.NewBlock()}

	sl.Block.Border = false

	sl.tv = tv
	return sl
}

func (ti *TreeItem) prevItem() *TreeItem {
	if ti.prev == nil {
		return ti.parent
	}

	ti = ti.prev

	// find last child of previous sibling
	for ti != nil {
		if ti.child == nil || ti.folded {
			return ti
		}

		ti = ti.child
		for ti.next != nil {
			ti = ti.next
		}
	}
	return nil
}

func (ti *TreeItem) nextItem() *TreeItem {
	if ti.child == nil || ti.folded {
		for ti != nil {
			if ti.next != nil {
				return ti.next
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

	for c := ti.child; c != nil; c = c.next {
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

		node := ti.node.(*DepsNode)
		indent := 3 * node.depth
		cs := tui.DefaultTxBuilder.Build(node.name, fg, bg)
		cs = tui.DTrimTxCls(cs, tv.cols+2-indent)

		j := 0
		if i == tv.idx {
			// draw current line cursor from the beginning
			for j < indent {
				if j+1 > tv.pos {
					buf.Set(j+1-tv.pos, printed+1, tui.Cell{' ', fg, bg})
				}
				j++
			}
		} else {
			j = indent
		}

		if j+1 > tv.pos {
			if ti.folded {
				buf.Set(j+1-tv.pos, printed+1, tui.Cell{'+', fg, bg})
			} else {
				buf.Set(j+1-tv.pos, printed+1, tui.Cell{'-', fg, bg})
			}
		}
		if j+2 > tv.pos {
			buf.Set(j+2-tv.pos, printed+1, tui.Cell{' ', fg, bg})
		}
		j += 2

		for _, vv := range cs {
			w := vv.Width()
			if j+1 > tv.pos {
				buf.Set(j+1-tv.pos, printed+1, vv)
			}
			j += w
		}

		printed++
		i++

		if i != tv.idx+1 {
			continue
		}

		// draw current line cursor to the end
		for j < tv.cols+tv.pos {
			if j+1 > tv.pos {
				buf.Set(j+1-tv.pos, printed, tui.Cell{' ', fg, bg})
			}
			j++
		}
	}

	return buf
}

func (tv *TreeView) Down() {
	if tv.idx < tv.Root.total {
		tv.idx++
		tv.Curr = tv.Curr.nextItem()
	}
	if tv.idx-tv.off >= tv.rows {
		tv.off++
		tv.Top = tv.Top.nextItem()
	}
}

func (tv *TreeView) Up() {
	if tv.idx > 0 {
		tv.idx--
		tv.Curr = tv.Curr.prevItem()
	}
	if tv.idx < tv.off {
		tv.off = tv.idx
		tv.Top = tv.Curr
	}
}

func (tv *TreeView) PageDown() {
	idx := tv.idx

	bottom := tv.off + tv.rows - 1
	if bottom > tv.Root.total {
		bottom = tv.Root.total
	}

	// At first, move to the bottom of current page
	if tv.idx != bottom {
		tv.idx = bottom

		for idx != bottom {
			tv.Curr = tv.Curr.nextItem()
			idx++
		}
		return
	}

	tv.idx += tv.rows
	if tv.idx > tv.Root.total {
		tv.idx = tv.Root.total
	}

	for idx != tv.idx {
		tv.Curr = tv.Curr.nextItem()
		idx++
	}

	off := tv.off

	if tv.idx-tv.off >= tv.rows {
		tv.off = tv.idx - tv.rows + 1

		for off != tv.off {
			tv.Top = tv.Top.nextItem()
			off++
		}
	}
}

func (tv *TreeView) PageUp() {
	idx := tv.idx

	// At first, move to the top of current page
	if tv.idx != tv.off {
		tv.idx = tv.off
		tv.Curr = tv.Top
		return
	}

	tv.idx -= tv.rows
	if tv.idx < 0 {
		tv.idx = 0
	}

	tv.off = tv.idx

	for idx != tv.idx {
		tv.Curr = tv.Curr.prevItem()
		idx--
	}

	tv.Top = tv.Curr
}

func (tv *TreeView) Home() {
	tv.idx = 0
	tv.off = 0
	tv.Curr = tv.Root
	tv.Top = tv.Root
}

func (tv *TreeView) End() {
	tv.idx = tv.Root.total
	tv.off = tv.idx - tv.rows + 1

	if tv.off < 0 {
		tv.off = 0
	}

	for next := tv.Curr; next != nil; next = next.nextItem() {
		tv.Curr = next
	}

	off := tv.idx
	top := tv.Curr

	for off != tv.off {
		top = top.prevItem()
		off--
	}

	tv.Top = top
}

func (tv *TreeView) Left(i int) {
	tv.pos -= i
	if tv.pos < 0 {
		tv.pos = 0
	}
}

func (tv *TreeView) Right(i int) {
	tv.pos += i
}

func (tv *TreeView) Toggle() {
	tv.Curr.toggle()
}

// Buffer implements Bufferer interface.
func (sl *StatusLine) Buffer() tui.Buffer {
	buf := sl.Block.Buffer()

	var line string

	curr := sl.tv.Curr
	if curr != nil {
		node := curr.node.(*DepsNode)
		line = node.name

		n := node.parent
		for n != nil {
			line = n.name + " > " + line

			n = n.parent
		}
	} else {
		line = "ELF tree"
	}

	fg := tui.ColorBlack
	bg := tui.ColorWhite

	cs := tui.DefaultTxBuilder.Build(line, fg, bg)
	cs = tui.DTrimTxCls(cs, sl.Width-3)

	buf.Set(0, sl.Y, tui.Cell{' ', fg, bg})
	buf.Set(1, sl.Y, tui.Cell{' ', fg, bg})

	j := 2
	for _, vv := range cs {
		w := vv.Width()
		buf.Set(j, sl.Y, vv)
		j += w
	}

	// draw status line to the end
	for j < sl.Width {
		buf.Set(j, sl.Y, tui.Cell{' ', fg, bg})
		j++
	}

	return buf
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

	tv.Height = tui.TermHeight() - 1
	tv.Width = tui.TermWidth()
	tv.Root = root
	tv.Curr = root
	tv.Top = root

	tv.rows = tv.Height - 2 // exclude border at top and bottom
	tv.cols = tv.Width - 2  // exclude border at left and right

	tui.Render(tv)

	sl := NewStatusLine(tv)
	sl.Height = 1
	sl.Width = tui.TermWidth()
	sl.Y = tui.TermHeight() - 1

	tui.Render(sl)

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
		tui.Render(sl)
	})
	tui.Handle("/sys/kbd/<up>", func(tui.Event) {
		tv.Up()
		tui.Render(tv)
		tui.Render(sl)
	})
	tui.Handle("/sys/kbd/<left>", func(tui.Event) {
		tv.Left(1)
		tui.Render(tv)
		// no need to redraw sl
	})
	tui.Handle("/sys/kbd/<right>", func(tui.Event) {
		tv.Right(1)
		tui.Render(tv)
		// no need to redraw sl
	})
	tui.Handle("/sys/kbd/<", func(tui.Event) {
		tv.Left(3)
		tui.Render(tv)
		// no need to redraw sl
	})
	tui.Handle("/sys/kbd/>", func(tui.Event) {
		tv.Right(3)
		tui.Render(tv)
		// no need to redraw sl
	})
	tui.Handle("/sys/kbd/<next>", func(tui.Event) {
		tv.PageDown()
		tui.Render(tv)
		tui.Render(sl)
	})
	tui.Handle("/sys/kbd/<previous>", func(tui.Event) {
		tv.PageUp()
		tui.Render(tv)
		tui.Render(sl)
	})
	tui.Handle("/sys/kbd/<home>", func(tui.Event) {
		tv.Home()
		tui.Render(tv)
		tui.Render(sl)
	})
	tui.Handle("/sys/kbd/<end>", func(tui.Event) {
		tv.End()
		tui.Render(tv)
		tui.Render(sl)
	})

	tui.Handle("/sys/kbd/<enter>", func(tui.Event) {
		tv.Toggle()
		tui.Render(tv)
		tui.Render(sl)
	})

	tui.Handle("/sys/wnd/resize", func(tui.Event) {
		tv.Height = tui.TermHeight() - 1
		tv.Width = tui.TermWidth()
		tv.rows = tv.Height - 2
		tv.cols = tv.Width - 2
		tui.Render(tv)

		sl.Width = tui.TermWidth()
		sl.Y = tui.TermHeight() - 1
		tui.Render(sl)
	})

	tui.Loop()
}
