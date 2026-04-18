package models

import (
	"sort"
	"sync"
	"time"
)

var (
	clickData     map[string][]URLClick
	clickDataMu   sync.RWMutex
	dailyStats    map[string]map[string]int
	dailyStatsMu  sync.RWMutex
)

type Analytics struct {
	TotalClicks     int64                      `json:"total_clicks"`
	UniqueClicks    int64                      `json:"unique_clicks"`
	DailyStats      map[string]int             `json:"daily_stats"`
	TopReferrers    []ReferrerStats            `json:"top_referrers"`
	TopUserAgents   []UserAgentStats           `json:"top_user_agents"`
	RecentClicks    []URLClick                 `json:"recent_clicks"`
	ClicksByCountry map[string]int             `json:"clicks_by_country"`
}

type ReferrerStats struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
}

type UserAgentStats struct {
	UserAgent string `json:"user_agent"`
	Count     int    `json:"count"`
}

func init() {
	clickData = make(map[string][]URLClick)
	dailyStats = make(map[string]map[string]int)
}

func ClearAnalyticsData() {
	clickDataMu.Lock()
	defer clickDataMu.Unlock()
	clickData = make(map[string][]URLClick)

	dailyStatsMu.Lock()
	defer dailyStatsMu.Unlock()
	dailyStats = make(map[string]map[string]int)
}

func LogClick(shortCode, userAgent, ipAddress, referer string) {
	clickDataMu.Lock()
	defer clickDataMu.Unlock()

	click := URLClick{
		ID:        generateID(),
		ShortCode: shortCode,
		Timestamp: time.Now(),
		UserAgent: userAgent,
		IPAddress: ipAddress,
		Referer:   referer,
	}

	clickData[shortCode] = append(clickData[shortCode], click)

	today := time.Now().Format("2006-01-02")
	dailyStatsMu.Lock()
	if dailyStats[shortCode] == nil {
		dailyStats[shortCode] = make(map[string]int)
	}
	dailyStats[shortCode][today]++
	dailyStatsMu.Unlock()

	if len(clickData[shortCode]) > 1000 {
		clickData[shortCode] = clickData[shortCode][len(clickData[shortCode])-1000:]
	}
}

func GetAnalytics(shortCode string) Analytics {
	clickDataMu.RLock()
	defer clickDataMu.RUnlock()

	clicks := clickData[shortCode]
	if clicks == nil {
		return Analytics{
			DailyStats:      make(map[string]int),
			TopReferrers:    []ReferrerStats{},
			TopUserAgents:   []UserAgentStats{},
			RecentClicks:    []URLClick{},
			ClicksByCountry: make(map[string]int),
		}
	}

	analytics := Analytics{
		TotalClicks:     int64(len(clicks)),
		UniqueClicks:    getUniqueClicks(clicks),
		DailyStats:      getDailyStats(shortCode),
		TopReferrers:    getTopReferrers(clicks),
		TopUserAgents:   getTopUserAgents(clicks),
		RecentClicks:    getRecentClicks(clicks),
		ClicksByCountry: getClicksByCountry(clicks),
	}

	return analytics
}

func getUniqueClicks(clicks []URLClick) int64 {
	uniqueIPs := make(map[string]bool)
	for _, click := range clicks {
		uniqueIPs[click.IPAddress] = true
	}
	return int64(len(uniqueIPs))
}

func getDailyStats(shortCode string) map[string]int {
	dailyStatsMu.RLock()
	defer dailyStatsMu.RUnlock()

	stats := make(map[string]int)
	if dailyStats[shortCode] != nil {
		for date, count := range dailyStats[shortCode] {
			stats[date] = count
		}
	}
	return stats
}

func getTopReferrers(clicks []URLClick) []ReferrerStats {
	referrers := make(map[string]int)
	for _, click := range clicks {
		if click.Referer != "" {
			referrers[click.Referer]++
		} else {
			referrers["Direct"]++
		}
	}

	var stats []ReferrerStats
	for ref, count := range referrers {
		stats = append(stats, ReferrerStats{
			Referrer: ref,
			Count:    count,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	if len(stats) > 10 {
		stats = stats[:10]
	}

	return stats
}

func getTopUserAgents(clicks []URLClick) []UserAgentStats {
	userAgents := make(map[string]int)
	for _, click := range clicks {
		if click.UserAgent != "" {
			userAgents[click.UserAgent]++
		}
	}

	var stats []UserAgentStats
	for ua, count := range userAgents {
		stats = append(stats, UserAgentStats{
			UserAgent: ua,
			Count:     count,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	if len(stats) > 10 {
		stats = stats[:10]
	}

	return stats
}

func getRecentClicks(clicks []URLClick) []URLClick {
	if len(clicks) <= 20 {
		return clicks
	}
	return clicks[len(clicks)-20:]
}

func getClicksByCountry(clicks []URLClick) map[string]int {
	countries := make(map[string]int)
	for _, click := range clicks {
		country := getCountryFromIP(click.IPAddress)
		countries[country]++
	}
	return countries
}

func getCountryFromIP(ip string) string {
	if ip == "127.0.0.1" || ip == "::1" {
		return "Local"
	}
	return "Unknown"
}