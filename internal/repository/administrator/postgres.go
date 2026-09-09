package administrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/administrator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// adminColumns joins auth.users for the email and aliases per-table fields
// so the resulting scan lines up with the Administrator struct.
const adminColumns = `p.id, COALESCE(p.full_name, ''), u.email::text, COALESCE(p.institution, ''), COALESCE(p.occupation, ''), p.roles::text, p.is_active, p.must_change_password, p.application_id, p.created_at`

func (r *PostgresRepository) Create(ctx context.Context, a *domain.Administrator) error {
	_, err := r.db.Exec(ctx, `INSERT INTO public.profiles
        (id, full_name, institution, occupation, roles, application_id, is_active, must_change_password)
        VALUES ($1, NULLIF($2,''), NULLIF($3,''), NULLIF($4,''), 'administrator', $5, $6, true)`,
		a.ID, a.FullName, a.Institution, a.Occupation, a.ApplicationID, a.IsActive)
	if err != nil {
		return fmt.Errorf("create administrator profile: %w", err)
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, f domain.ListFilter) ([]domain.Administrator, int, error) {
	page := f.Page
	size := f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	where := "p.roles::text = 'administrator'"
	args := []any{}
	if f.IsActive != nil {
		args = append(args, *f.IsActive)
		where += fmt.Sprintf(" AND p.is_active = $%d", len(args))
	}
	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM public.profiles p JOIN auth.users u ON u.id = p.id WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count administrators: %w", err)
	}
	args = append(args, size, (page-1)*size)
	limitArg := fmt.Sprintf("$%d", len(args)-1)
	offsetArg := fmt.Sprintf("$%d", len(args))
	query := `SELECT ` + adminColumns + `
        FROM public.profiles p
        JOIN auth.users u ON u.id = p.id
        WHERE ` + where + `
        ORDER BY p.created_at DESC LIMIT ` + limitArg + ` OFFSET ` + offsetArg
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list administrators: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Administrator, 0)
	for rows.Next() {
		a, err := scanAdministrator(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate administrators: %w", err)
	}
	return items, total, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*domain.Administrator, error) {
	row := r.db.QueryRow(ctx, `SELECT `+adminColumns+`
        FROM public.profiles p JOIN auth.users u ON u.id = p.id
        WHERE p.id = $1 AND p.roles::text = 'administrator'`, id)
	a, err := scanAdministrator(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get administrator: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.Administrator, error) {
	setClauses := ""
	args := []any{id}
	argIdx := 2
	if in.FullName != nil {
		setClauses += fmt.Sprintf("full_name = $%d", argIdx)
		args = append(args, *in.FullName)
		argIdx++
	}
	if in.IsActive != nil {
		setClauses += fmt.Sprintf("is_active = $%d", argIdx)
		args = append(args, *in.IsActive)
		argIdx++
	}
	if setClauses == "" {
		return r.Get(ctx, id)
	}
	query := fmt.Sprintf("UPDATE public.profiles SET %s WHERE id = $1 AND roles::text = 'administrator' RETURNING %s", setClauses, adminColumns)
	row := r.db.QueryRow(ctx, query, args...)
	a, err := scanAdministrator(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update administrator: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM public.profiles WHERE id = $1 AND roles::text = 'administrator'`, id)
	if err != nil {
		return fmt.Errorf("delete administrator: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanAdministrator(row rowScanner) (domain.Administrator, error) {
	var a domain.Administrator
	var institution, occupation sql.NullString
	if err := row.Scan(&a.ID, &a.FullName, &a.Email, &institution, &occupation, &a.Role, &a.IsActive, &a.MustChangePassword, &a.ApplicationID, &a.CreatedAt); err != nil {
		return a, err
	}
	if institution.Valid {
		a.Institution = institution.String
	}
	if occupation.Valid {
		a.Occupation = occupation.String
	}
	return a, nil
}
