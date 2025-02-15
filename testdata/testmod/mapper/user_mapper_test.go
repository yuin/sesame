package mapper_test

import (
	"context"
	"testing"

	"example.com/testmod/domain"
	"example.com/testmod/mapper"
	. "example.com/testmod/mapper"
	"example.com/testmod/model"
	"github.com/google/go-cmp/cmp"
)

func TestUserNammer(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringConverter(mappers)
	ctx := context.TODO()

	userMapper, err := NewTypedMappers[UserMapper](mappers).Get("UserMapper")
	if err != nil {
		t.Fatal(err)
	}

	source := &model.UserModel{
		ID:        "id1",
		Name:      "name1",
		UpdatedAt: "2024-07-18T10:15:36Z",
		Address: &model.AddressModel{
			Pref: "Tokyo",
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
			Pref: "Tokyo",
		},
	}
	if diff := cmp.Diff(expected, &entity); len(diff) != 0 {
		t.Errorf("Compare value is mismatch(-:expected, +:actual) :%s\n", diff)
	}
}
