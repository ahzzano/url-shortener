package api

import (
	"net/http"
	"url-shortener/services"
	"url-shortener/services/rerouter"
	"url-shortener/services/shortener"
)

type App struct {
    services []services.Service
}

func New() App {
    return App {
        services: []services.Service{
            rerouter.New(),
            shortener.New(),
        },

    }
}

func (s *App) Mount() {
    for _, service := range s.services {
        service.Mount()
    }
}

func (s *App) Run() {
    http.ListenAndServe(":8080", nil)
}
