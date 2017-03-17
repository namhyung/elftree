package main

import tui "github.com/gizak/termui"

type ScrollList struct {
	tui.Block    // embedded
	Items        []string
	ItemFgColor  tui.Attribute
	ItemBgColor  tui.Attribute
	FocusFgColor tui.Attribute
	FocusBgColor tui.Attribute

	idx int
	off int
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

// Buffer implements Bufferer interface.
func (l *ScrollList) Buffer() tui.Buffer {
	buf := l.Block.Buffer()

	printed := 0

	for i, v := range l.Items {
		if i < l.off {
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
		}

		cs := tui.DefaultTxBuilder.Build(v, fg, bg)
		cs = tui.DTrimTxCls(cs, l.Width-2)

		j := 0
		for _, vv := range cs {
			w := vv.Width()
			buf.Set(j+1, printed+1, vv)
			j += w
		}

		printed++

		if i != l.idx {
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
	if l.idx < len(l.Items)-1 {
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

func makeItems(dep *DepsNode, items []string) []string {
	str := ""
	for i := 0; i < dep.depth; i++ {
		str += "   "
	}
	str += dep.name

	if showPath {
		str += " => " + deps[dep.name].path
	}

	items = append(items, str)
	for _, v := range dep.child {
		items = makeItems(v, items)
	}
	return items
}

func ShowWithTUI(dep *DepsNode) {
	if err := tui.Init(); err != nil {
		panic(err)
	}
	defer tui.Close()

	items := makeItems(dep, []string{})

	ls := NewScrollList()

	ls.BorderLabel = "Tree view"
	ls.Height = tui.TermHeight()
	ls.Width = tui.TermWidth()
	ls.Items = items

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

	tui.Handle("/sys/wnd/resize", func(tui.Event) {
		ls.Height = tui.TermHeight()
		ls.Width = tui.TermWidth()
		tui.Render(ls)
	})

	tui.Loop()
}
