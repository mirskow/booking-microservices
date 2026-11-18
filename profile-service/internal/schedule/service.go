package schedule

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid schedule data")
	ErrUserNotFound = errors.New("user not found")
)

const maxScheduleRules = 100

type UserProvider interface {
	GetTimezone(ctx context.Context, userID uuid.UUID) (string, error)
}

type Service struct {
	repo         Repository
	users        UserProvider
	slotDuration time.Duration
}

func NewService(
	repo Repository,
	users UserProvider,
	slotDuration time.Duration,
) *Service {
	return &Service{
		repo:         repo,
		users:        users,
		slotDuration: slotDuration,
	}
}

// SetSchedule полностью заменяет расписание пользователя.
func (s *Service) SetSchedule(
	ctx context.Context,
	userID uuid.UUID,
	req SetScheduleRequest,
) ([]AvailabilityRule, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user ID", ErrInvalidInput)
	}

	if err := validateRules(req.Rules); err != nil {
		return nil, err
	}

	// Проверяем существование пользователя.
	_, err := s.users.GetTimezone(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user timezone: %w", err)
	}

	rules, err := s.repo.ReplaceRules(ctx, userID, req.Rules)
	if err != nil {
		return nil, fmt.Errorf("replace schedule: %w", err)
	}

	return rules, nil
}

// GetSchedule возвращает регулярное расписание пользователя.
func (s *Service) GetSchedule(
	ctx context.Context,
	userID uuid.UUID,
) ([]AvailabilityRule, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user ID", ErrInvalidInput)
	}

	_, err := s.users.GetTimezone(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user timezone: %w", err)
	}

	rules, err := s.repo.GetRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	return rules, nil
}

// GetSlots генерирует доступные временные слоты на указанную дату.
func (s *Service) GetSlots(
	ctx context.Context,
	userID uuid.UUID,
	date string,
) (*SlotsResponse, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: invalid user ID", ErrInvalidInput)
	}

	if s.slotDuration <= 0 {
		return nil, fmt.Errorf("slot duration must be positive")
	}

	// Получаем часовой пояс пользователя.
	timezone, err := s.users.GetTimezone(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user timezone: %w", err)
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load user timezone: %w", err)
	}

	// Разбираем дату именно в часовом поясе пользователя.
	requestedDate, err := time.ParseInLocation(
		"2006-01-02",
		date,
		location,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: date must have format YYYY-MM-DD",
			ErrInvalidInput,
		)
	}

	// Не принимаем даты с некорректным календарным значением.
	if requestedDate.Format("2006-01-02") != date {
		return nil, fmt.Errorf(
			"%w: invalid calendar date",
			ErrInvalidInput,
		)
	}

	rules, err := s.repo.GetRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get schedule: %w", err)
	}

	// Go: Sunday = 0, Monday = 1, ..., Saturday = 6.
	// В нашей модели: Monday = 0, ..., Sunday = 6.
	weekday := (int(requestedDate.Weekday()) + 6) % 7

	slots := make([]Slot, 0)

	for _, rule := range rules {
		if rule.Weekday != weekday {
			continue
		}

		startClock, err := time.Parse("15:04:05", rule.StartTime)
		if err != nil {
			return nil, fmt.Errorf("parse schedule start time: %w", err)
		}

		endClock, err := time.Parse("15:04:05", rule.EndTime)
		if err != nil {
			return nil, fmt.Errorf("parse schedule end time: %w", err)
		}

		start := time.Date(
			requestedDate.Year(),
			requestedDate.Month(),
			requestedDate.Day(),
			startClock.Hour(),
			startClock.Minute(),
			0,
			0,
			location,
		)

		end := time.Date(
			requestedDate.Year(),
			requestedDate.Month(),
			requestedDate.Day(),
			endClock.Hour(),
			endClock.Minute(),
			0,
			0,
			location,
		)

		for slotStart := start; ; {
			slotEnd := slotStart.Add(s.slotDuration)

			if slotEnd.After(end) {
				break
			}

			slots = append(slots, Slot{
				Start: slotStart,
				End:   slotEnd,
			})

			slotStart = slotEnd
		}
	}

	sort.Slice(slots, func(i, j int) bool {
		return slots[i].Start.Before(slots[j].Start)
	})

	return &SlotsResponse{
		UserID:   userID,
		Date:     date,
		Timezone: timezone,
		Slots:    slots,
	}, nil
}

// validateRules проверяет корректность всего расписания.
func validateRules(rules []AvailabilityRuleInput) error {
	if len(rules) > maxScheduleRules {
		return fmt.Errorf(
			"%w: maximum number of rules is %d",
			ErrInvalidInput,
			maxScheduleRules,
		)
	}

	// Группируем интервалы по дням недели.
	byWeekday := make(map[int][]AvailabilityRuleInput)

	for i, rule := range rules {
		if rule.Weekday < 0 || rule.Weekday > 6 {
			return fmt.Errorf(
				"%w: rule %d: weekday must be between 0 and 6",
				ErrInvalidInput,
				i,
			)
		}

		start, err := parseClockTime(rule.StartTime)
		if err != nil {
			return fmt.Errorf(
				"%w: rule %d: invalid start_time",
				ErrInvalidInput,
				i,
			)
		}

		end, err := parseClockTime(rule.EndTime)
		if err != nil {
			return fmt.Errorf(
				"%w: rule %d: invalid end_time",
				ErrInvalidInput,
				i,
			)
		}

		if !start.Before(end) {
			return fmt.Errorf(
				"%w: rule %d: start_time must be before end_time",
				ErrInvalidInput,
				i,
			)
		}

		byWeekday[rule.Weekday] = append(
			byWeekday[rule.Weekday],
			rule,
		)
	}

	// Проверяем пересечения интервалов внутри каждого дня.
	for weekday, dayRules := range byWeekday {
		sort.Slice(dayRules, func(i, j int) bool {
			return dayRules[i].StartTime < dayRules[j].StartTime
		})

		for i := 1; i < len(dayRules); i++ {
			previousEnd, _ := parseClockTime(dayRules[i-1].EndTime)
			currentStart, _ := parseClockTime(dayRules[i].StartTime)

			if currentStart.Before(previousEnd) {
				return fmt.Errorf(
					"%w: overlapping intervals on weekday %d",
					ErrInvalidInput,
					weekday,
				)
			}
		}
	}

	return nil
}

// parseClockTime принимает строгое время в формате HH:MM.
func parseClockTime(value string) (time.Time, error) {
	t, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}, err
	}

	if t.Format("15:04") != value {
		return time.Time{}, fmt.Errorf("time must use HH:MM format")
	}

	return t, nil
}
