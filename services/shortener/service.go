package shortener

import (
	"context"
	"encoding/base64"
	"math/big"
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

	bigVal := big.NewInt(id)
	bytes := bigVal.Bytes()

	if len(bytes) == 0 {
		bytes = []byte{0}
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}
