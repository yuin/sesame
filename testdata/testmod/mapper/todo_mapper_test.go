package mapper_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"example.com/testmod/domain"
	"example.com/testmod/mapper"
	. "example.com/testmod/mapper"
	"example.com/testmod/model"
)

func mustTime(s string) time.Time {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestTodoMapper(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringMapper(mappers)

	obj, err := mappers.Get("TodoMapper")
	if err != nil {
		t.Fatal(err)
	}
	todoMapper, _ := obj.(TodoMapper)

	source := &model.TodoModel{
		ID:     1,
		UserID: "AAA",
		Title:  "Write unit tests",
		Type:   1,
		Attributes: map[string][]string{
			"Date":     []string{"20240101", "20240130"},
			"Priority": []string{"High"},
		},
		Tags:         [5]string{"Task"},
		Done:         false,
		UpdatedAt:    "2023-07-18T10:15:36Z",
		ValidateOnly: true,
	}

	entity, err := todoMapper.TodoModelToTodo(source)
	if err != nil {
		t.Fatal(err)
	}
	expected := &domain.Todo{
		ID: 1,
		User: &domain.User{
			ID: "AAA",
		},
		Title: "Write unit tests",
		Type:  domain.TodoTypePrivate,
		Attributes: map[string][]string{
			"Date":     []string{"20240101", "20240130"},
			"Priority": []string{"High"},
		},
		Tags:      [5]string{"Task"},
		Finished:  false,
		UpdatedAt: mustTime("2023-07-18T10:15:36Z"),
	}

	if diff := cmp.Diff(expected, entity); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}

	// entity.ID=in64, model.ID=int(32), so ID can not be casted into a dest type
	entity.ID = 1
	reversed, err := todoMapper.TodoToTodoModel(entity)
	if err != nil {
		t.Fatal(err)
	}
	source.ValidateOnly = false
	source.ID = 0

	if diff := cmp.Diff(source, reversed); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}
}

type todoMapperHelper struct {
}

var _ TodoMapperHelper = &todoMapperHelper{}

func (h *todoMapperHelper) TodoModelToTodo(source *model.TodoModel, dest *domain.Todo) error {
	if source.ValidateOnly {
		dest.Attributes["ValidateOnly"] = []string{"true"}
	}
	return nil
}

func (h *todoMapperHelper) TodoToTodoModel(source *domain.Todo, dest *model.TodoModel) error {
	if _, ok := source.Attributes["ValidateOnly"]; ok {
		dest.ValidateOnly = true
	}
	return nil
}

func TestMapperHelper(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringMapper(mappers)

	mappers.AddFactory("TodoMapperHelper", func(ms Mappers) (any, error) {
		return &todoMapperHelper{}, nil
	})

	obj, err := mappers.Get("TodoMapper")
	if err != nil {
		t.Fatal(err)
	}
	todoMapper, _ := obj.(TodoMapper)

	source := &model.TodoModel{
		ID:     1,
		UserID: "AAA",
		Title:  "Write unit tests",
		Type:   1,
		Attributes: map[string][]string{
			"Date":     []string{"20240101", "20240130"},
			"Priority": []string{"High"},
		},
		Tags:         [5]string{"Task"},
		Done:         false,
		UpdatedAt:    "2023-07-18T10:15:36Z",
		ValidateOnly: true,
	}

	entity, err := todoMapper.TodoModelToTodo(source)
	if err != nil {
		t.Fatal(err)
	}
	expected := &domain.Todo{
		ID: 1,
		User: &domain.User{
			ID: "AAA",
		},
		Title: "Write unit tests",
		Type:  domain.TodoTypePrivate,
		Attributes: map[string][]string{
			"Date":         []string{"20240101", "20240130"},
			"Priority":     []string{"High"},
			"ValidateOnly": []string{"true"},
		},
		Tags:      [5]string{"Task"},
		Finished:  false,
		UpdatedAt: mustTime("2023-07-18T10:15:36Z"),
	}

	if diff := cmp.Diff(expected, entity); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}

}
