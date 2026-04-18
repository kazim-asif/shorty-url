package utils

import (
	"crypto/rand"
	"math/big"
	"net/url"
	"regexp"
)

const (
	DefaultLength = 6
	MaxLength     = 10
	MinLength     = 4
)

var (
	base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	urlRegex    = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
)

type Shortener struct {
	Length int
}

func NewShortener() *Shortener {
	return &Shortener{
		Length: DefaultLength,
	}
}

func (s *Shortener) SetLength(length int) {
	if length < MinLength {
		s.Length = MinLength
	} else if length > MaxLength {
		s.Length = MaxLength
	} else {
		s.Length = length
	}
}

func (s *Shortener) GenerateShortCode() (string, error) {
	result := make([]byte, s.Length)
	for i := 0; i < s.Length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[num.Int64()]
	}
	return string(result), nil
}

func ValidateURL(rawURL string) (string, error) {
	if !urlRegex.MatchString(rawURL) {
		return "", &URLValidationError{URL: rawURL}
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", &URLValidationError{URL: rawURL, Err: err}
	}

	return parsedURL.String(), nil
}

type URLValidationError struct {
	URL string
	Err error
}

func (e *URLValidationError) Error() string {
	if e.Err != nil {
		return "invalid URL: " + e.URL + " - " + e.Err.Error()
	}
	return "invalid URL format: " + e.URL
}
