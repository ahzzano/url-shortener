package rerouter

import (
	"context"
	"strconv"
	queries "url-shortener/database/postgres"
)

type RerouterService struct {
	database queries.Queries
}

func (s *RerouterService) GetURL(ctx context.Context, id string) (string, error) {
	idInt, err := strconv.ParseInt(id, 36, 64)
	if err != nil {
		return "", err
	}

	url, err := s.database.GetURLByID(ctx, idInt)
	if err != nil {
		return "", err
	}

	return url, nil
}
