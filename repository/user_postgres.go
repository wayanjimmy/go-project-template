package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-project-template/database/sqldb"
	"go-project-template/entity"
	"go-project-template/logger"
	"go-project-template/service"

	sq "github.com/Masterminds/squirrel"
)

type PostgresUserRepository struct {
	db        *sqldb.DB
	sb        sq.StatementBuilderType
	encryptor Encryptor
	log       *logger.Logger
}

var _ service.UserRepository = (*PostgresUserRepository)(nil)

func NewPostgresUserRepository(db *sqldb.DB, encryptor Encryptor, log *logger.Logger) *PostgresUserRepository {
	if log == nil {
		log = logger.Noop()
	}

	return &PostgresUserRepository{
		db:        db,
		sb:        sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		encryptor: encryptor,
		log:       log,
	}
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	r.log.Info(ctx, "repository.user.find_by_id", "user_id", id)
	query, args, err := r.sb.
		Select("id", "name", "email", "address_encrypted").
		From("users").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find user query failed: %w", err)
	}

	row := r.db.SQL().QueryRowContext(ctx, query, args...)

	var user entity.User
	var encryptedAddress []byte
	if err := row.Scan(&user.ID, &user.Name, &user.Email, &encryptedAddress); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user %s not found", id)
		}
		return nil, fmt.Errorf("find user by id failed: %w", err)
	}

	address, err := r.encryptor.Decrypt(ctx, encryptedAddress, []byte(user.ID))
	if err != nil {
		return nil, fmt.Errorf("decrypt address failed: %w", err)
	}
	user.Address = address

	return &user, nil
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *entity.User) error {
	r.log.Info(ctx, "repository.user.save", "user_id", user.ID)
	encryptedAddress, err := r.encryptor.Encrypt(ctx, user.Address, []byte(user.ID))
	if err != nil {
		return fmt.Errorf("encrypt address failed: %w", err)
	}

	query, args, err := r.sb.
		Insert("users").
		Columns("id", "name", "email", "address_encrypted").
		Values(user.ID, user.Name, user.Email, encryptedAddress).
		Suffix(`ON CONFLICT (id)
		DO UPDATE SET
			name = EXCLUDED.name,
			email = EXCLUDED.email,
			address_encrypted = EXCLUDED.address_encrypted,
			updated_at = NOW()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save user query failed: %w", err)
	}

	if _, err := r.db.SQL().ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("save user failed: %w", err)
	}

	return nil
}

func (r *PostgresUserRepository) List(ctx context.Context, limit, offset int) ([]entity.User, error) {
	r.log.Info(ctx, "repository.user.list", "limit", limit, "offset", offset)

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query, args, err := r.sb.
		Select("id", "name", "email", "address_encrypted").
		From("users").
		OrderBy("updated_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list users query failed: %w", err)
	}

	rows, err := r.db.SQL().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users failed: %w", err)
	}
	defer rows.Close()

	users := make([]entity.User, 0, limit)
	for rows.Next() {
		var user entity.User
		var encryptedAddress []byte
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &encryptedAddress); err != nil {
			return nil, fmt.Errorf("scan user failed: %w", err)
		}

		address, err := r.encryptor.Decrypt(ctx, encryptedAddress, []byte(user.ID))
		if err != nil {
			r.log.Error(ctx, "repository.user.list.decrypt_error", "user_id", user.ID, "error", err.Error())
			user.Address = "<error decrypting>"
		} else {
			user.Address = address
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users failed: %w", err)
	}

	return users, nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	r.log.Info(ctx, "repository.user.delete", "user_id", id)
	query, args, err := r.sb.
		Delete("users").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete user query failed: %w", err)
	}

	if _, err := r.db.SQL().ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("delete user failed: %w", err)
	}
	return nil
}
