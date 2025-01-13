package mapper

import (
	"context"
	"time"

	"example.com/testmod/domain"
)

type TimeStringConverter struct {
}

func (m *TimeStringConverter) StringToTime(ctx context.Context, source *string) (*time.Time, error) {
	if source == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *source)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (m *TimeStringConverter) TimeToString(ctx context.Context, source *time.Time) (string, bool, error) {
	if source == nil {
		return "", true, nil
	}
	return source.Format(time.RFC3339), false, nil
}

func AddTimeToStringConverter(mappers interface {
	Add(string, any)
}) {
	stringTime := &TimeStringConverter{}
	mappers.Add("TimeStringConverter", stringTime)
}

type InfStringConverter struct {
}

func (m *InfStringConverter) StringToInf(ctx context.Context, source *string) (domain.Inf, error) {
	if source == nil {
		return &domain.InfV{}, nil
	}
	return &domain.InfV{*source}, nil
}

func (m *InfStringConverter) InfToString(ctx context.Context, source domain.Inf) (string, bool, error) {
	if source == nil {
		return "", true, nil
	}
	return source.Value(), false, nil
}

func AddInfToStringConverter(mappers interface {
	Add(string, any)
}) {
	stringInf := &InfStringConverter{}
	mappers.Add("InfStringConverter", stringInf)
}
