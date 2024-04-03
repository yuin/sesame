package mapper

import (
	"context"
	"time"
)

type TimeStringMapper struct {
}

func (m *TimeStringMapper) StringToTime(ctx context.Context, source string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, source)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (m *TimeStringMapper) TimeToString(ctx context.Context, source *time.Time) (string, error) {
	return source.Format(time.RFC3339), nil
}

func AddTimeToStringMapper(mappers interface {
	Add(string, any)
}) {
	stringTime := &TimeStringMapper{}
	mappers.Add("TimeStringMapper", stringTime)
}
