package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	v1 "go-project-template/rest/v1"
)

type Options struct {
	OnboardingHandler *v1.UserOnboardingHandler
	WorkflowDiag      http.Handler
}

type Option func(*Options)

func WithUserOnboardingHandler(h *v1.UserOnboardingHandler) Option {
	return func(o *Options) {
		o.OnboardingHandler = h
	}
}

func WithWorkflowDiagnostics(h http.Handler) Option {
	return func(o *Options) {
		o.WorkflowDiag = h
	}
}

func applyOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func mountOptionalRoutes(r chi.Router, options Options) {
	if options.OnboardingHandler != nil {
		r.Route("/v1", func(r chi.Router) {
			r.Post("/users/onboarding", options.OnboardingHandler.Start)
			r.Post("/users/{id}/verify-email", options.OnboardingHandler.VerifyEmail)
		})
	}

	if options.WorkflowDiag != nil {
		r.Handle("/diag/workflows/*", http.StripPrefix("/diag/workflows", options.WorkflowDiag))
	}
}
