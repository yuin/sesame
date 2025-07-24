package domain

import "time"

type TodoType int

const (
	TodoTypeUnknown TodoType = iota
	TodoTypePrivate
	TodoTypeWork
)

type Priority string

type Todo struct {
	ID           int64
	Type         TodoType
	User         *User
	Title        string
	Attributes   map[string][]string
	Tags         [5]string
	Priorities   []Priority
	Finished     bool
	UpdatedAt    time.Time
	CreatedAt    time.Time
	Inf          Inf
	privateValue int
}

func (e *Todo) SetPrivateValue(v int) {
	e.privateValue = v
}

func (e *Todo) PrivateValue() int {
	return e.privateValue
}

type Inf interface {
	Value() string
}

type InfV struct {
	Valuef string
}

func (i *InfV) Value() string {
	return i.Valuef
}
