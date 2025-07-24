package model

type TodoModel struct {
	Id           int
	UserID       string
	UserAddress  *AddressModel
	Title        string
	Type         int
	Attributes   map[string][]string
	Tags         [5]string
	Priorities   string
	Done         bool
	UpdatedAt    string
	CreatedAt    string
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
