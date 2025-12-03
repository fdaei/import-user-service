package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"rankr/app/service/user"
	"rankr/pkg/database"
	types "rankr/type"
)

type userRepository struct {
	db *database.Database
}

// NewUserRepository creates a repository for user data backed by PostgreSQL.
func NewUserRepository(db *database.Database) user.Repository {
	return &userRepository{db: db}
}

func (r *userRepository) UpsertUser(ctx context.Context, u user.User) error {
	rawID, err := parseUserID(u.ID)
	if err != nil {
		return err
	}

	tx, err := r.db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	const insertUser = `
		INSERT INTO users (id, name, email, phone_number, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    email = EXCLUDED.email,
		    phone_number = EXCLUDED.phone_number,
		    updated_at = EXCLUDED.updated_at
	`

	if _, err := tx.Exec(ctx, insertUser,
		rawID,
		u.Name,
		u.Email,
		u.PhoneNumber,
		u.CreatedAt,
		u.UpdatedAt,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM addresses WHERE user_id = $1`, rawID); err != nil {
		return err
	}

	if len(u.Addresses) > 0 {
		batch := &pgx.Batch{}
		for _, addr := range u.Addresses {
			batch.Queue(`
				INSERT INTO addresses (user_id, street, city, state, zip_code, country, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`,
				rawID,
				addr.Street,
				addr.City,
				addr.State,
				addr.ZipCode,
				addr.Country,
				now,
				now,
			)
		}

		results := tx.SendBatch(ctx, batch)
		for i := 0; i < len(u.Addresses); i++ {
			if _, err := results.Exec(); err != nil {
				results.Close()
				return err
			}
		}
		if err := results.Close(); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *userRepository) GetByID(ctx context.Context, id types.ID) (user.User, error) {
	const query = `
		SELECT u.id,
		       u.name,
		       u.email,
		       u.phone_number,
		       u.created_at,
		       u.updated_at,
		       COALESCE(
				   json_agg(
					   json_build_object(
						   'id', a.id,
						   'street', a.street,
						   'city', a.city,
						   'state', a.state,
						   'zip_code', a.zip_code,
						   'country', a.country,
						   'created_at', a.created_at,
						   'updated_at', a.updated_at
					   )
					   ORDER BY a.id
				   ) FILTER (WHERE a.id IS NOT NULL),
				   '[]'
			   ) AS addresses
		FROM users u
		LEFT JOIN addresses a ON a.user_id = u.id
		WHERE u.id = $1
		GROUP BY u.id
	`

	var (
		rawID     uint64
		addrBytes []byte
		res       user.User
	)

	if err := r.db.Pool.QueryRow(ctx, query, uint64(id)).Scan(
		&rawID,
		&res.Name,
		&res.Email,
		&res.PhoneNumber,
		&res.CreatedAt,
		&res.UpdatedAt,
		&addrBytes,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}
		return user.User{}, err
	}

	res.ID = types.ID(rawID)
	if len(addrBytes) > 0 {
		if err := json.Unmarshal(addrBytes, &res.Addresses); err != nil {
			return user.User{}, fmt.Errorf("decode addresses: %w", err)
		}
	}

	return res, nil
}

func parseUserID(id types.ID) (uint64, error) {
	val := uint64(id)
	if val == 0 {
		return 0, errors.New("id must be greater than zero")
	}

	return val, nil
}
