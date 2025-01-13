package model

type TodoModel struct {
	Id           int
	UserID       string
	Title        string
	Type         int
	Attributes   map[string][]string
	Tags         [5]string
	Done         bool
	UpdatedAt    string
	ValidateOnly bool
	Inf          string
	privateValue int
}

func (m *TodoModel) SetPrivateValue(v int) {
	m.privateValue = v
}

func (m *TodoModel) PrivateValue() int {
	return m.privateValue
}
