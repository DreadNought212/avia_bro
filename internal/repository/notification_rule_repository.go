package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type NotificationRule struct {
	ID              int
	RouteID         int
	MaxPrice        int
	TelegramEnabled bool
}

type NotificationRuleRepository struct {
	db *sql.DB
}

func NewNotificationRuleRepository(db *sql.DB) *NotificationRuleRepository {
	return &NotificationRuleRepository{
		db: db,
	}
}

func (r *NotificationRuleRepository) GetByRouteID(
	ctx context.Context,
	routeID int,
) (*NotificationRule, error) {

	query := `
		SELECT
			id,
			route_id,
			max_price,
			telegram_enabled
		FROM common.notification_rules
		WHERE route_id = $1
	`

	var rule NotificationRule

	err := r.db.QueryRowContext(
		ctx,
		query,
		routeID,
	).Scan(
		&rule.ID,
		&rule.RouteID,
		&rule.MaxPrice,
		&rule.TelegramEnabled,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"failed to get notification rule for route %d: %w",
			routeID,
			err,
		)
	}

	return &rule, nil
}
