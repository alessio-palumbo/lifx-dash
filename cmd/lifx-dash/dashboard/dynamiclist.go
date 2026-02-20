package dashboard

import (
	"fyne.io/fyne/v2"
)

type Renderable interface {
	CanvasObject() fyne.CanvasObject
	comparable
}

type DynamicList[T Renderable] struct {
	Box      *fyne.Container
	MaxItems int

	items    []T
	OnChange func(items []T)
}

func NewDynamicList[T Renderable](box *fyne.Container, max int) *DynamicList[T] {
	return &DynamicList[T]{
		Box:      box,
		MaxItems: max,
	}
}

func (l *DynamicList[T]) Add(item T) bool {
	if len(l.items) >= l.MaxItems {
		return false
	}
	l.items = append(l.items, item)
	l.Box.Add(item.CanvasObject())
	l.changed()
	return true
}

func (l *DynamicList[T]) Remove(item T) {
	for i, it := range l.items {
		if it == item {
			l.items = append(l.items[:i], l.items[i+1:]...)
			break
		}
	}
	l.Box.Remove(item.CanvasObject())
	l.changed()
}

func (l *DynamicList[T]) Items() []T {
	return l.items
}

func (l *DynamicList[T]) Count() int {
	return len(l.items)
}

func (l *DynamicList[T]) IsFull() bool {
	return len(l.items) >= l.MaxItems
}

func (l *DynamicList[T]) IndexOf(item T) int {
	for i, it := range l.items {
		if it == item {
			return i
		}
	}
	return -1
}

func (l *DynamicList[T]) changed() {
	if l.OnChange != nil {
		l.OnChange(l.items)
	}
}
