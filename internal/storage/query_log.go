package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"blockads-desktop/internal/models"
)

// QueryEvent represents a single DNS resolution event to be logged.
type QueryEvent struct {
	ID                string
	Timestamp         time.Time
	Domain            string
	ClientIP          string
	QueryType         string
	Status            string // "ALLOWED", "BLOCKED", "FAILED"
	UpstreamLatencyMs int
	FilterRule        string
	Reason            string
	Protocol          string
	Rcode             int
}

// QueryFilter defines criteria for filtering and paginating query events.
type QueryFilter struct {
	Domain    string
	Status    string
	ClientIP  string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// InsertQueryEventsBatch bulk-inserts a batch of query events inside a single transaction.
func (s *Storage) InsertQueryEventsBatch(ctx context.Context, events []QueryEvent) error {
	if len(events) == 0 {
		return nil
	}

	return s.InTransaction(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO query_events (
				id, timestamp, domain, client_ip, query_type, status,
				upstream_latency_ms, filter_rule, reason, protocol, rcode
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare query_events insert: %w", err)
		}
		defer stmt.Close()

		for i, ev := range events {
			id := ev.ID
			if id == "" {
				id = fmt.Sprintf("ev-%d-%d", ev.Timestamp.UnixNano(), i)
			}
			tsStr := ev.Timestamp.UTC().Format(time.RFC3339Nano)

			_, err := stmt.ExecContext(ctx,
				id,
				tsStr,
				strings.ToLower(strings.TrimSpace(ev.Domain)),
				strings.TrimSpace(ev.ClientIP),
				strings.ToUpper(strings.TrimSpace(ev.QueryType)),
				strings.ToUpper(strings.TrimSpace(ev.Status)),
				ev.UpstreamLatencyMs,
				strings.TrimSpace(ev.FilterRule),
				strings.TrimSpace(ev.Reason),
				strings.ToUpper(strings.TrimSpace(ev.Protocol)),
				ev.Rcode,
			)
			if err != nil {
				return fmt.Errorf("failed to insert query event %s: %w", id, err)
			}
		}

		return nil
	})
}

// GetQueryEvents returns paginated query log entries matching the filter criteria.
func (s *Storage) GetQueryEvents(ctx context.Context, filter QueryFilter) ([]models.QueryLogEntry, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var whereClauses []string
	var args []any

	if filter.Domain != "" {
		whereClauses = append(whereClauses, "domain LIKE ?")
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Domain))+"%")
	}
	if filter.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Status)))
	}
	if filter.ClientIP != "" {
		whereClauses = append(whereClauses, "client_ip = ?")
		args = append(args, strings.TrimSpace(filter.ClientIP))
	}
	if filter.StartTime != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, filter.StartTime.UTC().Format(time.RFC3339Nano))
	}
	if filter.EndTime != nil {
		whereClauses = append(whereClauses, "timestamp <= ?")
		args = append(args, filter.EndTime.UTC().Format(time.RFC3339Nano))
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Total count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM query_events %s;", whereSQL)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count query_events: %w", err)
	}

	// 2. Paginated data query
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, timestamp, domain, client_ip, query_type, status,
		       upstream_latency_ms, filter_rule, reason
		FROM query_events
		%s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?;
	`, whereSQL)

	queryArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query query_events: %w", err)
	}
	defer rows.Close()

	var entries []models.QueryLogEntry
	for rows.Next() {
		var e models.QueryLogEntry
		err := rows.Scan(
			&e.ID,
			&e.Timestamp,
			&e.Domain,
			&e.ClientIP,
			&e.QueryType,
			&e.Status,
			&e.UpstreamLatencyMs,
			&e.FilterRule,
			&e.Reason,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan query_event: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// PruneQueryEvents deletes historical events older than the specified duration.
func (s *Storage) PruneQueryEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339Nano)

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, "DELETE FROM query_events WHERE timestamp < ?;", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to prune query_events: %w", err)
	}

	return res.RowsAffected()
}

// PruneMaxQueryEvents retains at most maxCount most recent events, deleting excess records.
func (s *Storage) PruneMaxQueryEvents(ctx context.Context, maxCount int) (int64, error) {
	if maxCount <= 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		DELETE FROM query_events
		WHERE id NOT IN (
			SELECT id FROM query_events
			ORDER BY timestamp DESC
			LIMIT ?
		);
	`

	res, err := s.db.ExecContext(ctx, query, maxCount)
	if err != nil {
		return 0, fmt.Errorf("failed to prune excess query_events: %w", err)
	}

	return res.RowsAffected()
}
