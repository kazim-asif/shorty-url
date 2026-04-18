package models

import (
	"errors"
	"sync"
	"time"
)

var (
	URLs map[string]*URL
	mu   sync.RWMutex
)

type URL struct {
	ID          string    `json:"id"`
	OriginalURL string    `json:"original_url"`
	ShortCode   string    `json:"short_code"`
	Clicks      int64     `json:"clicks"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool      `json:"is_active"`
	UserAgent   string    `json:"user_agent,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

type URLClick struct {
	ID        string    `json:"id"`
	ShortCode string    `json:"short_code"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Referer   string    `json:"referer"`
}

func init() {
	URLs = make(map[string]*URL)
}

func CreateURL(originalURL, shortCode, userAgent, ipAddress string) (*URL, error) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := URLs[shortCode]; exists {
		return nil, errors.New("short code already exists")
	}

	url := &URL{
		ID:          generateID(),
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		Clicks:      0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsActive:    true,
		UserAgent:   userAgent,
		IPAddress:   ipAddress,
	}

	URLs[shortCode] = url
	return url, nil
}

func GetURLByShortCode(shortCode string) (*URL, error) {
	mu.RLock()
	defer mu.RUnlock()

	if url, ok := URLs[shortCode]; ok {
		if !url.IsActive {
			return nil, errors.New("URL is deactivated")
		}
		if url.ExpiresAt != nil && url.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("URL has expired")
		}
		return url, nil
	}
	return nil, errors.New("short code not found")
}

func IncrementClicks(shortCode string) error {
	mu.Lock()
	defer mu.Unlock()

	if url, ok := URLs[shortCode]; ok {
		url.Clicks++
		url.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("short code not found")
}

func GetAllURLs() map[string]*URL {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]*URL)
	for k, v := range URLs {
		result[k] = v
	}
	return result
}

func DeleteURL(shortCode string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := URLs[shortCode]; ok {
		delete(URLs, shortCode)
		return nil
	}
	return errors.New("short code not found")
}

func generateID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(result)
}

