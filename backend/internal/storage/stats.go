package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepo struct {
	pool *pgxpool.Pool
}

func NewStatsRepo(pool *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{pool: pool}
}

type EventCount struct {
	Event string
	Count int
}

type DayCount struct {
	Day   time.Time
	Count int
}

type DashboardStats struct {
	UsersTotal       int
	UsersVerified    int
	UsersPremium     int
	UsersAdmin       int
	Signups7d        int
	Signups30d       int
	ReadingsTotal    int
	Readings7d       int
	Readings30d      int
	TxPaid           int
	TxRefunded       int
	RevenuePaidKop   int64
	RevenueRefundKop int64
	SignupsByDay     []DayCount
	Events30d        []EventCount
}

func (r *StatsRepo) Dashboard(ctx context.Context) (*DashboardStats, error) {
	s := &DashboardStats{
		SignupsByDay: make([]DayCount, 0),
		Events30d:    make([]EventCount, 0),
	}

	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE email_verified_at IS NOT NULL),
			COUNT(*) FILTER (WHERE has_premium),
			COUNT(*) FILTER (WHERE role = 'admin'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days')
		FROM users`).Scan(
		&s.UsersTotal, &s.UsersVerified, &s.UsersPremium, &s.UsersAdmin, &s.Signups7d, &s.Signups30d,
	)
	if err != nil {
		return nil, fmt.Errorf("user stats: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days')
		FROM readings`).Scan(&s.ReadingsTotal, &s.Readings7d, &s.Readings30d)
	if err != nil {
		return nil, fmt.Errorf("reading stats: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'paid'),
			COUNT(*) FILTER (WHERE status = 'refunded'),
			COALESCE(SUM(amount_kop) FILTER (WHERE status = 'paid'), 0),
			COALESCE(SUM(amount_kop) FILTER (WHERE status = 'refunded'), 0)
		FROM transactions`).Scan(&s.TxPaid, &s.TxRefunded, &s.RevenuePaidKop, &s.RevenueRefundKop)
	if err != nil {
		return nil, fmt.Errorf("billing stats: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT (created_at AT TIME ZONE 'UTC')::date AS d, COUNT(*)
		FROM users
		WHERE created_at >= NOW() - INTERVAL '14 days'
		GROUP BY 1
		ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("signups by day: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d DayCount
		if err = rows.Scan(&d.Day, &d.Count); err != nil {
			return nil, err
		}
		s.SignupsByDay = append(s.SignupsByDay, d)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	erows, err := r.pool.Query(ctx, `
		SELECT event, COUNT(*)
		FROM access_stats
		WHERE created_at >= NOW() - INTERVAL '30 days'
		GROUP BY event
		ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, fmt.Errorf("event stats: %w", err)
	}
	defer erows.Close()
	for erows.Next() {
		var e EventCount
		if err = erows.Scan(&e.Event, &e.Count); err != nil {
			return nil, err
		}
		s.Events30d = append(s.Events30d, e)
	}
	return s, erows.Err()
}

func (r *StatsRepo) CountReadingsForUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM readings WHERE user_id = $1`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count user readings: %w", err)
	}
	return n, nil
}
