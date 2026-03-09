package dashboard

const (
	inactive = -1
)

type CommandHistory struct {
	items []string
	index int
	max   int
}

func NewCommandHistory(max int) *CommandHistory {
	return &CommandHistory{
		max:   max,
		index: inactive,
	}
}

func (h *CommandHistory) Add(cmd string) {
	if cmd == "" {
		return
	}

	if len(h.items) > 0 && h.items[len(h.items)-1] == cmd {
		return
	}

	h.items = append(h.items, cmd)

	if len(h.items) > h.max {
		h.items = h.items[1:]
	}

	h.index = inactive
}

func (h *CommandHistory) Prev() (string, bool) {
	if len(h.items) == 0 {
		return "", false
	}

	if h.index == -1 {
		h.index = len(h.items) - 1
	} else if h.index > 0 {
		h.index--
	}

	return h.items[h.index], true
}

func (h *CommandHistory) Next() (string, bool) {
	if h.index == inactive {
		return "", false
	}

	if h.index < len(h.items)-1 {
		h.index++
		return h.items[h.index], true
	}

	h.index = inactive
	return "", true
}

func (h *CommandHistory) Active() bool {
	return h.index != inactive
}
