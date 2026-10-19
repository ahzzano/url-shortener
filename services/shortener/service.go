package shortener

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	queries "url-shortener/database/postgres"
)

var ErrInvalidURL = errors.New("Invalid URL")

type ShortenerService struct {
	queries *queries.Queries
}

func (s *ShortenerService) validateURL(rawURL string) bool {
	if strings.ContainsFunc(rawURL, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Hostname() != "" && parsed.User == nil
}

func (s *ShortenerService) shortenURL(ctx context.Context, url string) (string, error) {
	if s.validateURL(url) == false {
		return "", ErrInvalidURL
	}

	id, err := s.queries.CreateURL(ctx, url)
	if err != nil {
		return "", err
	}

	shortened := strconv.FormatInt(id, 36)

	return shortened, nil
}
