package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

func (r *DashboardRepository) CountUsers(ctx context.Context) (int, error) {
	return r.count(ctx, "users", "TRUE")
}

func (r *DashboardRepository) CountProducts(ctx context.Context) (int, error) {
	return r.count(ctx, "products", "active = TRUE")
}

func (r *DashboardRepository) CountApplications(ctx context.Context) (int, error) {
	return r.count(ctx, "applications", "TRUE")
}

func (r *DashboardRepository) CountConversations(ctx context.Context) (int, error) {
	return r.count(ctx, "conversations", "TRUE")
}

func (r *DashboardRepository) CountMessages(ctx context.Context) (int, error) {
	return r.count(ctx, "messages", "TRUE")
}

func (r *DashboardRepository) Series(ctx context.Context, from, to time.Time) ([]models.DashboardSeriesPoint, error) {
	userCounts, err := r.series(ctx, "users", from, to)
	if err != nil {
		return nil, err
	}
	productCounts, err := r.series(ctx, "products", from, to, "active = TRUE")
	if err != nil {
		return nil, err
	}
	applicationCounts, err := r.series(ctx, "applications", from, to)
	if err != nil {
		return nil, err
	}
	conversationCounts, err := r.series(ctx, "conversations", from, to)
	if err != nil {
		return nil, err
	}
	messageCounts, err := r.series(ctx, "messages", from, to)
	if err != nil {
		return nil, err
	}

	points := make([]models.DashboardSeriesPoint, 0)
	for day := from.Truncate(24 * time.Hour).UTC(); !day.After(to.UTC()); day = day.Add(24 * time.Hour) {
		key := day.Format("2006-01-02")
		points = append(points, models.DashboardSeriesPoint{
			Date:          key,
			Users:         userCounts[key],
			Products:      productCounts[key],
			Applications:  applicationCounts[key],
			Conversations: conversationCounts[key],
			Messages:      messageCounts[key],
		})
	}

	return points, nil
}

func (r *DashboardRepository) count(ctx context.Context, table, filter string) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, filter)
	var count int
	if err := r.pool.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return count, nil
}

func (r *DashboardRepository) series(ctx context.Context, table string, from, to time.Time, extra ...string) (map[string]int, error) {
	filter := "created_at >= $1 AND created_at < $2"
	if len(extra) > 0 && extra[0] != "" {
		filter = filter + " AND " + extra[0]
	}

	query := fmt.Sprintf(`
		SELECT DATE_TRUNC('day', created_at)::date AS day, COUNT(*)
		FROM %s
		WHERE %s
		GROUP BY day
		ORDER BY day
	`, table, filter)

	rows, err := r.pool.Query(ctx, query, from.UTC(), to.UTC().Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("query %s series: %w", table, err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("scan %s series: %w", table, err)
		}
		out[day.UTC().Format("2006-01-02")] = count
	}
	return out, nil
}
