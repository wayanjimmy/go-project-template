// Package v1 is the HTTP transport layer (API v1) built on chi.
//
// Handlers decode requests, validate input, call service interfaces, and
// render responses — including mapping apperror kinds to HTTP status codes.
// All business rules live in service; handlers stay thin.
//
// Dependency rule: v1 may import service (interfaces), entity, apperror,
// logger, and HTTP plumbing. It must not import repository or drivers,
// and it must not contain business logic.
package v1
