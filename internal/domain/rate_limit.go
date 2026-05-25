package domain

import (
	"time"
)

type RateLimitRule struct {
	ID             int64     `json:"id"`
	Scope          string    `json:"scope"`
	Key            string    `json:"key"`
	LimitPerMinute *int      `json:"limit_per_minute,omitempty"`
	LimitPerHour   *int      `json:"limit_per_hour,omitempty"`
	LimitPerDay    *int      `json:"limit_per_day,omitempty"`
	Enabled        bool      `json:"enabled"`
	Description    *string   `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const (
	RateLimitScopeGlobal = "global"
	RateLimitScopeUser   = "user"
	RateLimitScopeIP     = "ip"
	RateLimitScopeRoom   = "room"
)
