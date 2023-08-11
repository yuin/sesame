package domain

import "time"

type TodoType int

const (
	TodoTypeUnknown TodoType = iota
	TodoTypePrivate
	TodoTypeWork
)

type Todo struct {
	ID           int64
	Type         TodoType
	User         *User
	Title        string
	Attributes   map[string][]string
	Tags         [5]string
	Finished     bool
	UpdatedAt    time.Time
	privateValue int
}

func (e *Todo) SetPrivateValue(v int) {
	e.privateValue = v
}

func (e *Todo) PrivateValue() int {
	return e.privateValue
}
