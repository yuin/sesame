package mapper_test

import (
	"context"
	"errors"
	"testing"

	"example.com/testmod/domain"
	"example.com/testmod/mapper"
	. "example.com/testmod/mapper"
	"example.com/testmod/model"
	"github.com/google/go-cmp/cmp"
	"github.com/yuin/sesame"
)

func TestMappersGetFunc(t *testing.T) {
	mappers := NewMappers()
	mapper.AddTimeToStringConverter(mappers)
	mapper.AddInfToStringConverter(mappers)
	mapper.AddStreetConverter(mappers)
	mapper.AddDate1Converter(mappers)
	ctx := context.TODO()

	_, err := sesame.GetMapperFunc[*model.Date1, *domain.Date1](mappers, "")
	var se *sesame.Error
	if errors.As(err, &se) && se.IsConverter() {
		c, err := sesame.GetToObjectConverterFunc[*model.Date1, *domain.Date1](mappers, "")
		if err != nil {
			t.Fatal(err)
		}
		v1 := &model.Date1{
			Year: "2001",
		}
		v2, err := c(ctx, v1)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(&domain.Date1{
			Year: 2001,
		}, v2); len(diff) != 0 {
			t.Error(diff)
		}
	} else {
		t.Error("must be a converter")
	}

}
