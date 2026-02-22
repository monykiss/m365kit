// Package admin provides IT admin functionality for org-wide usage statistics.
package admin

import (
	"sort"
	"time"

	"github.com/klytics/m365kit/internal/audit"
)

// UsageStats holds aggregated usage statistics across users.
type UsageStats struct {
	ActiveUsers  int            `json:"active_users"`
	CommandCount int            `json:"command_count"`
	TopCommands  []CommandStat  `json:"top_commands"`
	TopUsers     []UserStat     `json:"top_users"`
	ErrorRate    float64        `json:"error_rate"`
	UserErrors   int            `json:"user_errors"`
	SystemErrors int            `json:"system_errors"`
}

// CommandStat represents a command usage count.
type CommandStat struct {
	Command string  `json:"command"`
	Count   int     `json:"count"`
	Pct     float64 `json:"pct"`
}

// UserStat represents a user activity count.
type UserStat struct {
	UserID string `json:"user_id"`
	Count  int    `json:"count"`
}

// StatsFilter controls which entries are included in stats.
type StatsFilter struct {
	Since   time.Time
	Until   time.Time
	UserID  string
	Command string
}

// AggregateStats computes usage statistics from audit log entries.
func AggregateStats(entries []audit.Entry, filter StatsFilter) *UsageStats {
	filtered := audit.FilterEntries(entries, filter.Since, filter.Until, filter.Command, filter.UserID)

	stats := &UsageStats{
		CommandCount: len(filtered),
	}

	if len(filtered) == 0 {
		return stats
	}

	cmdCounts := make(map[string]int)
	userCounts := make(map[string]int)
	var userErrors, sysErrors int

	for _, e := range filtered {
		cmdCounts[e.Command]++
		if e.UserID != "" {
			userCounts[e.UserID]++
		}
		switch e.ExitCode {
		case 1:
			userErrors++
		case 2:
			sysErrors++
		}
	}

	stats.ActiveUsers = len(userCounts)
	stats.UserErrors = userErrors
	stats.SystemErrors = sysErrors
	totalErrors := userErrors + sysErrors
	if stats.CommandCount > 0 {
		stats.ErrorRate = float64(totalErrors) / float64(stats.CommandCount) * 100
	}

	// Top commands sorted by count
	for cmd, count := range cmdCounts {
		pct := float64(count) / float64(stats.CommandCount) * 100
		stats.TopCommands = append(stats.TopCommands, CommandStat{
			Command: cmd, Count: count, Pct: pct,
		})
	}
	sort.Slice(stats.TopCommands, func(i, j int) bool {
		return stats.TopCommands[i].Count > stats.TopCommands[j].Count
	})

	// Top users sorted by count
	for user, count := range userCounts {
		stats.TopUsers = append(stats.TopUsers, UserStat{UserID: user, Count: count})
	}
	sort.Slice(stats.TopUsers, func(i, j int) bool {
		return stats.TopUsers[i].Count > stats.TopUsers[j].Count
	})

	return stats
}
