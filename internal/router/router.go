// Package router assembles the route tree and mounts handlers + middleware.
package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"

	"brewme/internal/handler"
	"brewme/internal/health"
	appmiddleware "brewme/internal/middleware"
)

// Router builds the HTTP router for the API server.
func Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Throttle(100))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/api/v1/health", health.ServerHealth)

	// Serve uploaded files (e.g. avatars) at /uploads/* from the local disk.
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", handler.Register)
			r.Post("/login", handler.Login)
			r.Get("/username-available", handler.CheckUsername)
			r.Post("/logout", appmiddleware.SessionMiddleware(handler.UserLogout))
		})

		r.Route("/creators", func(r chi.Router) {
			r.Get("/{username}", handler.GetCreatorProfile)
			r.Get("/{username}/supporters", handler.GetSupportersFeed)
			r.Get("/{username}/posts", handler.GetCreatorPublicPosts)
		})

		r.Route("/discover", func(r chi.Router) {
			r.Get("/", handler.GetAllCreator)
		})

		r.Route("/dashboard", func(r chi.Router) {
			r.Get("/stats", appmiddleware.SessionMiddleware(handler.DashboardStats))
			r.Get("/supporters", appmiddleware.SessionMiddleware(handler.LatestSupporters))
			r.Get("/supporters-list", appmiddleware.SessionMiddleware(handler.GetSupportersList))
			r.Post("/supporters/{id}/reply", appmiddleware.SessionMiddleware(handler.SubmitReply))
			r.Get("/posts", appmiddleware.SessionMiddleware(handler.GetDashboardPosts))
			r.Post("/posts", appmiddleware.SessionMiddleware(handler.CreatePost))
			r.Put("/posts/{id}", appmiddleware.SessionMiddleware(handler.UpdatePost))
			r.Delete("/posts/{id}", appmiddleware.SessionMiddleware(handler.DeletePost))
			r.Get("/earnings", appmiddleware.SessionMiddleware(handler.GetDashboardEarnings))
			r.Get("/memberships", appmiddleware.SessionMiddleware(handler.GetDashboardMemberships))
			r.Post("/memberships", appmiddleware.SessionMiddleware(handler.CreateDashboardMembershipTier))
			r.Put("/memberships/{id}", appmiddleware.SessionMiddleware(handler.UpdateDashboardMembershipTier))
			r.Delete("/memberships/{id}", appmiddleware.SessionMiddleware(handler.DeleteDashboardMembershipTier))
			r.Get("/settings", appmiddleware.SessionMiddleware(handler.GetDashboardSettings))
			r.Patch("/settings/profile", appmiddleware.SessionMiddleware(handler.UpdateDashboardProfile))
			r.Post("/settings/avatar", appmiddleware.SessionMiddleware(handler.UpdateDashboardAvatar))
			r.Patch("/settings/notifications", appmiddleware.SessionMiddleware(handler.UpdateDashboardNotifications))
			r.Put("/settings/goal", appmiddleware.SessionMiddleware(handler.UpdateDashboardGoal))
			r.Get("/settings/stripe/status", appmiddleware.SessionMiddleware(handler.GetDashboardStripeStatus))
			r.Post("/payouts", appmiddleware.SessionMiddleware(handler.RequestPayout))
		})

		r.Route("/profile", func(r chi.Router) {
			r.Get("/", appmiddleware.SessionMiddleware(handler.GetUserProfile))
		})

		r.Route("/category", func(r chi.Router) {
			r.Get("/", handler.GetAllCategory)
		})
	})

	return r
}
