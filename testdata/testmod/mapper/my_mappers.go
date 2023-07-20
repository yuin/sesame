package mapper

import "time"

type TimeStringMapper struct {
}

func (m *TimeStringMapper) StringToTime(source string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, source)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (m *TimeStringMapper) TimeToString(source *time.Time) (string, error) {
	return source.Format(time.RFC3339), nil
}

func AddTimeToStringMapper(mappers interface {
	AddFactory(string, func(MapperGetter) (any, error))
	AddMapperFuncFactory(string, string, func(MapperGetter) (any, error))
}) {
	stringTime := &TimeStringMapper{}
	mappers.AddFactory("TimeStringMapper", func(m MapperGetter) (any, error) {
		return stringTime, nil
	})
	mappers.AddMapperFuncFactory("string", "time#Time", func(m MapperGetter) (any, error) {
		return stringTime.StringToTime, nil
	})
	mappers.AddMapperFuncFactory("time#Time", "string", func(m MapperGetter) (any, error) {
		return stringTime.TimeToString, nil
	})
}
