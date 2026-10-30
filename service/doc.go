// Package service is the business-logic layer, built on top of the domain.
//
// This package owns:
//
//   - the business logic (use cases) of the application
//   - the interface contracts its dependencies must satisfy, defined from
//     the consumer side: UserRepository, UserIndexer, UserSearcher, etc.
//   - the Transaction abstraction for atomic work across repositories
//
// Interfaces are declared here, never in repository or other implementation
// packages — the layer that consumes a dependency defines its contract
// (see https://www.gobeyond.dev/packages-as-layers/). Implementations
// (repository/, rest/, worker/) must not add methods to these interfaces;
// new operations belong here.
//
// Dependency rule: service may import entity, apperror, transaction,
// event, and logger. It must not import repository, rest, web, worker,
// or any driver package (sqlx, redis, asynq, watermill).
package service
