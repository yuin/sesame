package model

type TodoModel struct {
	ID           int
	UserID       string
	Title        string
	Type         int
	Attributes   map[string][]string
	Tags         [5]string
	Done         bool
	UpdatedAt    string
	ValidateOnly bool
}
