package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/application"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository persists applications in Supabase Postgres.
type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

const applicationColumns = `id, full_name, email, institution, occupation, reason, status::text, reviewed_by, reviewed_at, created_at`

func (r *PostgresRepository) Create(ctx context.Context, a *domain.Application) error {
	_, err := r.db.Exec(ctx, `INSERT INTO public.administrator_applications
		(id, full_name, email, institution, occupation, reason, status, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), 'pending', $7)`,
		a.ID, a.FullName, a.Email, a.Institution, a.Occupation, a.Reason, a.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicatePending
		}
		return fmt.Errorf("create application: %w", err)
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, status domain.Status, page, size int) ([]domain.Application, int, error) {
	where, args := "", []any{}
	if status != "" {
		where = " WHERE status::text = $1"
		args = append(args, string(status))
	}
	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM public.administrator_applications"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}
	args = append(args, size, (page-1)*size)
	limitArg, offsetArg := "$1", "$2"
	if status != "" {
		limitArg, offsetArg = "$2", "$3"
	}
	rows, err := r.db.Query(ctx, "SELECT "+applicationColumns+" FROM public.administrator_applications"+where+" ORDER BY created_at DESC LIMIT "+limitArg+" OFFSET "+offsetArg, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Application, 0)
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate applications: %w", err)
	}
	return items, total, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*domain.Application, error) {
	row := r.db.QueryRow(ctx, "SELECT "+applicationColumns+" FROM public.administrator_applications WHERE id = $1", id)
	a, err := scanApplication(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) UpdateReview(ctx context.Context, id string, status domain.Status, note, reviewerID string) (*domain.Application, error) {
	row := r.db.QueryRow(ctx, "UPDATE public.administrator_applications SET status = $2, review_note = NULLIF($3, ''), reviewed_by = NULLIF($4, '')::uuid, reviewed_at = now() WHERE id = $1 AND status::text = 'pending' RETURNING "+applicationColumns, id, status, note, reviewerID)
	a, err := scanApplication(row)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := r.Get(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		if existing.Status != domain.StatusPending {
			return nil, ErrDuplicatePending
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("review application: %w", err)
	}
	return &a, nil
}

type rowScanner interface{ Scan(...any) error }

func scanApplication(row rowScanner) (domain.Application, error) {
	var a domain.Application
	var status string
	var institution, occupation, reason sql.NullString
	if err := row.Scan(&a.ID, &a.FullName, &a.Email, &institution, &occupation, &reason, &status, &a.ReviewedBy, &a.ReviewedAt, &a.CreatedAt); err != nil {
		return a, err
	}
	if institution.Valid {
		a.Institution = institution.String
	}
	if occupation.Valid {
		a.Occupation = occupation.String
	}
	if reason.Valid {
		a.Reason = reason.String
	}
	a.Status = domain.Status(status)
	return a, nil
}
