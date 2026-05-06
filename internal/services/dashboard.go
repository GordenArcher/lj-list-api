package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
)

type dashboardRepository interface {
	CountUsers(ctx context.Context) (int, error)
	CountProducts(ctx context.Context) (int, error)
	CountApplications(ctx context.Context) (int, error)
	CountConversations(ctx context.Context) (int, error)
	CountMessages(ctx context.Context) (int, error)
	Series(ctx context.Context, from, to time.Time) ([]models.DashboardSeriesPoint, error)
}

type DashboardService struct {
	repo dashboardRepository
	now  func() time.Time
}

func NewDashboardService(repo *repositories.DashboardRepository, cfg config.Config) *DashboardService {
	_ = cfg
	return &DashboardService{repo: repo, now: time.Now}
}

func (s *DashboardService) GetStats(ctx context.Context, rangeName, fromValue, toValue string) (*models.DashboardStats, error) {
	from, to, label, err := resolveDashboardRange(s.now().UTC(), rangeName, fromValue, toValue)
	if err != nil {
		return nil, err
	}

	users, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	products, err := s.repo.CountProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("count products: %w", err)
	}
	apps, err := s.repo.CountApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("count applications: %w", err)
	}
	convs, err := s.repo.CountConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("count conversations: %w", err)
	}
	messages, err := s.repo.CountMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("count messages: %w", err)
	}

	series, err := s.repo.Series(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("build dashboard series: %w", err)
	}

	return &models.DashboardStats{
		Range:         label,
		From:          from.Format("2006-01-02"),
		To:            to.Format("2006-01-02"),
		TotalUsers:    users,
		TotalProducts: products,
		TotalApps:     apps,
		TotalConvs:    convs,
		TotalMessages: messages,
		Series:        series,
	}, nil
}

func resolveDashboardRange(now time.Time, rangeName, fromValue, toValue string) (time.Time, time.Time, string, error) {
	now = now.UTC()
	end := truncateDay(now)
	start := end
	label := "today"

	switch strings.ToLower(strings.TrimSpace(rangeName)) {
	case "", "today":
		start = end
		label = "today"
	case "week", "last_week", "lastweek":
		start = end.AddDate(0, 0, -6)
		label = "week"
	case "month", "last_month", "lastmonth":
		start = end.AddDate(0, 0, -29)
		label = "month"
	case "custom":
		if strings.TrimSpace(fromValue) == "" || strings.TrimSpace(toValue) == "" {
			return time.Time{}, time.Time{}, "", fmt.Errorf("from and to are required for custom range")
		}
		parsedFrom, err := time.Parse("2006-01-02", strings.TrimSpace(fromValue))
		if err != nil {
			return time.Time{}, time.Time{}, "", fmt.Errorf("invalid from date")
		}
		parsedTo, err := time.Parse("2006-01-02", strings.TrimSpace(toValue))
		if err != nil {
			return time.Time{}, time.Time{}, "", fmt.Errorf("invalid to date")
		}
		start = truncateDay(parsedFrom.UTC())
		end = truncateDay(parsedTo.UTC())
		if end.Before(start) {
			return time.Time{}, time.Time{}, "", fmt.Errorf("to date must be after from date")
		}
		label = "custom"
	default:
		return time.Time{}, time.Time{}, "", fmt.Errorf("invalid range")
	}

	return start, end, label, nil
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
