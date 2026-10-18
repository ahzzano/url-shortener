package shortener

import (
	"context"
	"strconv"
	queries "url-shortener/database/postgres"
)

type ShortenerService struct {
	queries *queries.Queries
}

func (s *ShortenerService) shortenURL(ctx context.Context, url string) (string, error) {
	id, err := s.queries.CreateURL(ctx, url)
	if err != nil {
		return "", err
	}

	shortened := strconv.FormatInt(id, 36)

	return shortened, nil
}
