// Package repository implements the persistence layer: concrete
// implementations of the interfaces declared in service (UserRepository,
// UserIndexer, UserSearcher) backed by Postgres via sqlx/squirrel.
//
// Nothing in this package defines business interfaces — it only satisfies
// them. Repositories return apperror values and accept transaction.Transaction
// for atomic work, as declared by the service layer.
//
// Dependency rule: repository may import entity, service (interfaces only),
// apperror, transaction, logger, and its drivers (sqldb, sqlx, squirrel).
// It must not be imported by service, entity, or handlers.
package repository
