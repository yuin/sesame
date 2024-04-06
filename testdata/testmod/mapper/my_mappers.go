package mapper

import (
	"context"
	"time"
)

type TimeStringMapper struct {
}

func (m *TimeStringMapper) StringToTime(ctx context.Context, source string, dest *time.Time) error {
	t, err := time.Parse(time.RFC3339, source)
	if err != nil {
		return err
	}
	*dest = t
	return nil
}

func (m *TimeStringMapper) TimeToString(ctx context.Context, source *time.Time, dest *string) error {
	*dest = source.Format(time.RFC3339)
	return nil
}

func AddTimeToStringMapper(mappers interface {
	Add(string, any)
}) {
	stringTime := &TimeStringMapper{}
	mappers.Add("TimeStringMapper", stringTime)
}
