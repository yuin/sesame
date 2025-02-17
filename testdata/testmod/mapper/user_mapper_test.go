package mapper_test

import (
	"context"
	"testing"

	"example.com/testmod/domain"
	"example.com/testmod/mapper"
	. "example.com/testmod/mapper"
	"example.com/testmod/model"
	"github.com/google/go-cmp/cmp"
	"github.com/yuin/sesame"
)

func TestUserNammer(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringConverter(mappers)
	mapper.AddStreetConverter(mappers)
	mapper.AddIntStringConverter(mappers)
	ctx := context.TODO()

	userMapper, err := sesame.Get[UserMapper](mappers, "UserMapper")
	if err != nil {
		t.Fatal(err)
	}

	source := &model.UserModel{
		ID:        "id1",
		Name:      "name1",
		UpdatedAt: "2024-07-18T10:15:36Z",
		Address: &model.AddressModel{
			Pref:      "Tokyo",
			Street:    []int{1, 2, 3},
			IntValues: []int{10, 11, 12},
		},
	}
	var entity domain.User
	err = userMapper.UserModelToUser(ctx, source, &entity)
	if err != nil {
		t.Fatal(err)
	}
	expected := &domain.User{
		ID:        "id1",
		Name:      "name1",
		UpdatedAt: mustTime("2024-07-18T10:15:36Z"),
		Address: &domain.Address{
			Pref:         "Tokyo",
			Street:       "1-2-3",
			StringValues: []string{"10", "11", "12"},
		},
	}
	if diff := cmp.Diff(expected, &entity); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}
}
