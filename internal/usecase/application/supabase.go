package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	domain "github.com/itauq-golang/internal/domain/application"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SupabaseProvisioner creates an Auth user (admin create) and creates its profiles row.
// NOTE: This implementation creates the user with a fixed password ("12345678").
// This is intended for the requested dev behavior; consider changing to a secure flow for production.
type SupabaseProvisioner struct {
	URL, ServiceRoleKey string
	DB                  *pgxpool.Pool
	Client              *http.Client
}

func (p *SupabaseProvisioner) InviteAdministrator(ctx context.Context, a *domain.Application) (Administrator, error) {
	// Build admin create-user payload with a fixed password
	payloadMap := map[string]interface{}{
		"email":         a.Email,
		"password":      "12345678",
		"email_confirm": true,
		"user_metadata": map[string]string{"full_name": a.FullName},
	}
	payload, _ := json.Marshal(payloadMap)

	// Use the admin create user endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.URL, "/")+"/auth/v1/admin/users", bytes.NewReader(payload))
	if err != nil {
		return Administrator{}, err
	}
	req.Header.Set("apikey", p.ServiceRoleKey)
	req.Header.Set("Authorization", "Bearer "+p.ServiceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return Administrator{}, fmt.Errorf("create auth user: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Administrator{}, fmt.Errorf("create auth user: supabase returned %s: %s", resp.Status, string(bodyBytes))
	}

	var user struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		return Administrator{}, fmt.Errorf("decode created user: %w", err)
	}

	_, err = p.DB.Exec(ctx, `INSERT INTO public.profiles (id, full_name, occupation, institution, roles, application_id) VALUES ($1, $2, $3, $4, 'administrator', $5)`, user.ID, a.FullName, a.Occupation, a.Institution, a.ID)
	if err != nil {
		return Administrator{}, fmt.Errorf("create administrator profile: %w", err)
	}

	return Administrator{ID: user.ID, Email: user.Email, Role: "administrator"}, nil
}
