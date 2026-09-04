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
		r.Use(h.ipRateLimit)

		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.Refresh)
		r.Post("/auth/logout", h.Logout)
		r.Post("/auth/verify-email", h.VerifyEmail)
		r.Post("/auth/resend-verification", h.ResendVerification)
		r.Post("/auth/forgot-password", h.ForgotPassword)
		r.Post("/auth/reset-password", h.ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(h.authMiddleware)
			r.Use(h.userRateLimit)

			r.Get("/me", h.Me)
			r.Delete("/me", h.DeleteMe)

			r.Get("/readings", h.ListReadings)
			r.Post("/readings", h.CreateReading)
			r.Get("/readings/{id}", h.GetReading)
			r.Patch("/readings/{id}", h.PatchReading)
			r.Delete("/readings/{id}", h.DeleteReading)

			r.Post("/interpretations", h.Interpret)
			r.Get("/usage", h.GetUsage)
			r.Post("/usage/daily-card", h.ConsumeDailyCard)
			r.Post("/usage/ad-session", h.StartAdSession)
			r.Post("/billing/purchases", h.ReportPurchase)
			r.Post("/billing/subscription-status", h.SyncSubscriptionStatus)
			r.Post("/billing/checkout", h.CreateCheckout)
			r.Get("/billing/checkout/{id}", h.GetCheckout)
			r.Get("/me/decks", h.MyDecks)
		})

		r.Post("/billing/yookassa/webhook", h.YooKassaWebhook)

		r.Group(func(r chi.Router) {
			r.Use(h.optionalAuth)
			r.Get("/shop/decks", h.ShopListDecks)
			r.Get("/shop/decks/{slug}", h.ShopGetDeck)
		})
	})

	r.Get("/pay/go", h.PayGo)
	r.Get("/pay/return", h.PayReturn)

	r.Get("/media/decks/{slug}/{file}", h.ServeDeckMedia)

	r.Route("/admin", func(r chi.Router) {
		r.Get("/login", h.AdminLoginPage)
		r.Post("/login", h.AdminLogin)

		r.Group(func(r chi.Router) {
			r.Use(h.adminMiddleware)
			r.Get("/", h.AdminDashboard)
			r.Post("/logout", h.AdminLogout)
			r.Get("/users", h.AdminUsers)
			r.Get("/users/{id}", h.AdminUser)
			r.Post("/users/{id}/premium", h.AdminSetPremium)
			r.Get("/transactions", h.AdminTransactions)
			r.Post("/transactions", h.AdminCreateTransaction)
			r.Get("/transactions/{id}", h.AdminTransaction)
			r.Post("/transactions/{id}/refund", h.AdminRefund)
			r.Get("/decks", h.AdminDecks)
			r.Get("/decks/new", h.AdminDeckNew)
			r.Post("/decks", h.AdminDeckCreate)
			r.Get("/decks/{id}", h.AdminDeck)
			r.Post("/decks/{id}", h.AdminDeckUpdate)
			r.Post("/decks/{id}/import", h.AdminDeckImport)
			r.Post("/decks/{id}/grant", h.AdminDeckGrant)
			r.Post("/users/{id}/decks", h.AdminDeckGrantOnUser)
		})
	})

	return r
}
