package mapper_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"example.com/testmod/domain"
	"example.com/testmod/mapper"
	. "example.com/testmod/mapper"
	"example.com/testmod/model"
)

var todoModelIgnores = cmpopts.IgnoreUnexported(model.TodoModel{})
var todoEntityIgnores = cmpopts.IgnoreUnexported(domain.Todo{})

func mustTime(s string) time.Time {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestTodoMapper(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringConverter(mappers)
	mapper.AddInfToStringConverter(mappers)
	ctx := context.TODO()

	obj, err := mappers.Get("testdata.TodoMapper")
	if err != nil {
		t.Fatal(err)
	}
	todoMapper, _ := obj.(TodoMapper)

	source := &model.TodoModel{
		Id:     1,
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
		Inf:          "hoge",
		ValidateOnly: true,
	}
	source.SetPrivateValue(10)

	var entity domain.Todo
	err = todoMapper.TodoModelToTodo(ctx, source, &entity)
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
		Inf:       &domain.InfV{"hoge"},
	}
	expected.SetPrivateValue(10)

	if diff := cmp.Diff(expected, &entity, todoEntityIgnores); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}
	if expected.PrivateValue() != entity.PrivateValue() {
		t.Errorf("private fields with getter/setter must be mapped")
	}

	// entity.ID=in64, model.Id=int(32), so ID can not be casted into a dest type
	entity.ID = 1
	var reversed model.TodoModel
	err = todoMapper.TodoToTodoModel(ctx, &entity, &reversed)
	if err != nil {
		t.Fatal(err)
	}
	source.ValidateOnly = false
	source.Id = 0

	if diff := cmp.Diff(source, &reversed, todoModelIgnores); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}
	if source.PrivateValue() != reversed.PrivateValue() {
		t.Errorf("private fields with getter/setter must be mapped")
	}
}

type todoMapperHelper struct {
}

var _ TodoMapperHelper = &todoMapperHelper{}

func (h *todoMapperHelper) TodoModelToTodo(ctx context.Context, source *model.TodoModel, dest *domain.Todo) error {
	if source.ValidateOnly {
		if dest.Attributes == nil {
			dest.Attributes = map[string][]string{}
		}
		dest.Attributes["ValidateOnly"] = []string{"true"}
	}
	return nil
}

func (h *todoMapperHelper) TodoToTodoModel(ctx context.Context, source *domain.Todo, dest *model.TodoModel) error {
	if source.Attributes == nil {
		return nil
	}
	if _, ok := source.Attributes["ValidateOnly"]; ok {
		dest.ValidateOnly = true
	}
	return nil
}

func TestMapperHelper(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringConverter(mappers)
	ctx := context.TODO()

	mappers.Add("testdata.TodoMapperHelper", &todoMapperHelper{})

	obj, err := mappers.Get("testdata.TodoMapper")
	if err != nil {
		t.Fatal(err)
	}
	todoMapper, _ := obj.(TodoMapper)

	source := &model.TodoModel{
		Id:     1,
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

	var entity domain.Todo
	err = todoMapper.TodoModelToTodo(ctx, source, &entity)
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

	if diff := cmp.Diff(expected, &entity, todoEntityIgnores); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}

}

func TestNilCollection(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringConverter(mappers)
	ctx := context.TODO()

	obj, err := mappers.Get("TodoEmptyMapper")
	if err != nil {
		t.Fatal(err)
	}
	todoMapper, _ := obj.(TodoMapper)

	source := &model.TodoModel{
		Id:           1,
		UserID:       "AAA",
		Title:        "Write unit tests",
		Type:         1,
		Attributes:   nil,
		Tags:         [5]string{"Task"},
		Done:         false,
		UpdatedAt:    "2023-07-18T10:15:36Z",
		ValidateOnly: true,
	}

	var entity domain.Todo
	err = todoMapper.TodoModelToTodo(ctx, source, &entity)
	if err != nil {
		t.Fatal(err)
	}
	expected := &domain.Todo{
		ID:         0,
		User:       nil,
		Title:      "Write unit tests",
		Type:       domain.TodoTypePrivate,
		Attributes: map[string][]string{},
		Tags:       [5]string{"Task"},
		Finished:   false,
		UpdatedAt:  mustTime("2023-07-18T10:15:36Z"),
	}

	if diff := cmp.Diff(expected, &entity, todoEntityIgnores); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}

	source = &model.TodoModel{
		Id:     1,
		UserID: "AAA",
		Title:  "Write unit tests",
		Type:   1,
		Attributes: map[string][]string{
			"Priority": nil,
		},
		Tags:         [5]string{"Task"},
		Done:         false,
		UpdatedAt:    "2023-07-18T10:15:36Z",
		ValidateOnly: true,
	}

	entity = domain.Todo{}
	err = todoMapper.TodoModelToTodo(ctx, source, &entity)
	if err != nil {
		t.Fatal(err)
	}
	expected = &domain.Todo{
		ID:    0,
		User:  nil,
		Title: "Write unit tests",
		Type:  domain.TodoTypePrivate,
		Attributes: map[string][]string{
			"Priority": make([]string, 0),
		},
		Tags:      [5]string{"Task"},
		Finished:  false,
		UpdatedAt: mustTime("2023-07-18T10:15:36Z"),
	}

	if diff := cmp.Diff(expected, &entity, todoEntityIgnores); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}

}

func TestMapperNotFound(t *testing.T) {
	mappers := NewMappers()
	_, err := mappers.Get("Dummy")
	var merr interface {
		NotFound() bool
	}
	if !errors.As(err, &merr) || !merr.NotFound() {
		t.Errorf("error should be a not found error")
	}
}
