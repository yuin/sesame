package mapper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"example.com/testmod/domain"
	"example.com/testmod/model"
	"github.com/yuin/sesame"
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

type FixedTimeStringConverter struct {
}

func (m *FixedTimeStringConverter) StringToTime(ctx context.Context, source *string) (*time.Time, error) {
	t := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	return &t, nil
}

func (m *FixedTimeStringConverter) TimeToString(ctx context.Context, source *time.Time) (string, bool, error) {
	return "2021-01-01T00:00:00Z", false, nil
}

func AddTimeToStringConverter(mappers sesame.Mappers) {
	mappers.Add("TimeStringConverter", &TimeStringConverter{})
	sesame.AddFactory(mappers, "FixedTimeStringConverter",
		func(mg sesame.MapperGetter) (*FixedTimeStringConverter, error) {
			return &FixedTimeStringConverter{}, nil
		}, sesame.WithNoGlobals())
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

func AddInfToStringConverter(mappers sesame.Mappers) {
	stringInf := &InfStringConverter{}
	mappers.Add("InfStringConverter", stringInf)
}

type StreetConverter struct {
}

func (m *StreetConverter) StringToSlice(ctx context.Context, source *string) ([]int, error) {
	if source == nil {
		return nil, nil
	}
	parts := strings.Split(*source, "-")
	result := make([]int, len(parts))
	for i, part := range parts {
		result[i], _ = strconv.Atoi(part)
	}
	return result, nil
}

func (m *StreetConverter) SliceToString(ctx context.Context, source []int) (string, bool, error) {
	if source == nil {
		return "", true, nil
	}
	parts := make([]string, len(source))
	for i, part := range source {
		parts[i] = strconv.Itoa(part)
	}
	return strings.Join(parts, "-"), false, nil
}

func AddStreetConverter(mappers sesame.Mappers) {
	mappers.Add("StreetConverter", &StreetConverter{}, sesame.WithNoGlobals())
}

type IntStringConverter struct {
}

func (m *IntStringConverter) StringToInt(ctx context.Context, source *string) (int, bool, error) {
	if source == nil {
		return 0, true, nil
	}
	i, err := strconv.Atoi(*source)
	if err != nil {
		return 0, false, err
	}
	return i, false, nil
}

func (m *IntStringConverter) IntToString(ctx context.Context, source *int) (string, bool, error) {
	if source == nil {
		return "", true, nil
	}
	return strconv.Itoa(*source), false, nil
}

func AddIntStringConverter(mappers sesame.Mappers) {
	mappers.Add("IntStringConverter", &IntStringConverter{})
}

type Date1Converter struct {
}

func (m *Date1Converter) ModelToEntity(ctx context.Context, source *model.Date1) (*domain.Date1, error) {
	v, _ := strconv.Atoi(source.Year)
	return &domain.Date1{Year: v}, nil
}

func (m *Date1Converter) EntityToModel(ctx context.Context, source *domain.Date1) (*model.Date1, error) {
	return &model.Date1{Year: strconv.Itoa(source.Year)}, nil
}

func AddDate1Converter(mappers sesame.Mappers) {
	mappers.Add("Date1Converter", &Date1Converter{})
}

type PrioritiesStringConverter struct {
}

func (m *PrioritiesStringConverter) StringToSlice(ctx context.Context, source *string) ([]domain.Priority, error) {
	if source == nil {
		return nil, nil
	}
	parts := strings.Split(*source, ",")
	result := make([]domain.Priority, len(parts))
	for i, part := range parts {
		result[i] = domain.Priority(part)
	}
	return result, nil
}

func (m *PrioritiesStringConverter) SliceToString(ctx context.Context, source []domain.Priority) (string, bool, error) {
	if source == nil {
		return "", true, nil
	}
	parts := make([]string, len(source))
	for i, part := range source {
		parts[i] = string(part)
	}
	return strings.Join(parts, ","), false, nil
}

func AddPrioritiesConverter(mappers sesame.Mappers) {
	mappers.Add("PrioritiesStringConverter", &PrioritiesStringConverter{})
}
