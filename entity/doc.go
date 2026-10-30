// Package entity is the domain layer: the application's business data types
// and nothing else.
//
// Types here must be plain structs with primitive fields and time.Time —
// no imports of infrastructure packages (database, HTTP, Redis, loggers).
// JSON tags are allowed for convenience. Every other layer depends on this
// package; this package depends on none of them.
//
// Dependency rule: entity imports nothing from the application.
package entity
