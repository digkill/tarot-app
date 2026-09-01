package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(corsMiddleware(h.cfg.CORSAllowedOrigins))
	r.Use(bodyLimit)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware)

			r.Get("/me", h.Me)

			r.Get("/readings", h.ListReadings)
			r.Post("/readings", h.CreateReading)
			r.Get("/readings/{id}", h.GetReading)
			r.Patch("/readings/{id}", h.PatchReading)
			r.Delete("/readings/{id}", h.DeleteReading)

			r.Post("/interpretations", h.Interpret)
		})
	})

	return r
}
