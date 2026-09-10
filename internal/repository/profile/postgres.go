package profile

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/profile"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const profileColumns = `id, COALESCE(full_name, ''), COALESCE(occupation, ''), COALESCE(institution, ''), roles::text, must_change_password, created_at`

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Profile, error) {
	row := r.db.QueryRow(ctx, "SELECT "+profileColumns+" FROM public.profiles WHERE id = $1", id)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.Profile, error) {
	setClauses := ""
	args := []any{id}
	argIdx := 2

	if in.FullName != nil {
		setClauses += fmt.Sprintf("full_name = $%d", argIdx)
		args = append(args, *in.FullName)
		argIdx++
	}
	if in.Institution != nil {
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += fmt.Sprintf("institution = NULLIF($%d, '')", argIdx)
		args = append(args, *in.Institution)
		argIdx++
	}
	if in.Occupation != nil {
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += fmt.Sprintf("occupation = NULLIF($%d, '')", argIdx)
		args = append(args, *in.Occupation)
		argIdx++
	}

	if setClauses == "" {
		return r.GetByID(ctx, id)
	}

	query := fmt.Sprintf("UPDATE public.profiles SET %s WHERE id = $1 RETURNING %s", setClauses, profileColumns)
	row := r.db.QueryRow(ctx, query, args...)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) ClearMustChangePassword(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "UPDATE public.profiles SET must_change_password = false WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("clear must_change_password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanProfile(row rowScanner) (domain.Profile, error) {
	var p domain.Profile
	if err := row.Scan(&p.ID, &p.FullName, &p.Occupation, &p.Institution, &p.Role, &p.MustChangePassword, &p.CreatedAt); err != nil {
		return p, err
	}
	return p, nil
}
