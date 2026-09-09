package administrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// SupabaseInviter creates an auth user via Supabase's admin API and returns
// the new user id. Mirrors the application's SupabaseProvisioner flow but
// doesn't tie creation to an application_applications row.
type SupabaseInviter struct {
	URL            string
	ServiceRoleKey string
	Client         *http.Client
}

func (s *SupabaseInviter) Invite(ctx context.Context, fullName, email string) (string, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"email":         email,
		"password":      "12345678",
		"email_confirm": true,
		"user_metadata": map[string]string{"full_name": fullName},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(s.URL, "/")+"/auth/v1/admin/users",
		bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", s.ServiceRoleKey)
	req.Header.Set("Authorization", "Bearer "+s.ServiceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create auth user: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create auth user: supabase returned %s: %s", resp.Status, string(body))
	}
	var user struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", fmt.Errorf("decode created user: %w", err)
	}
	return user.ID, nil
}

// MemoryInviter is the dev/in-memory fallback so POST /administrators still
// works when DATABASE_URL / Supabase credentials are not configured.
type MemoryInviter struct{}

func (MemoryInviter) Invite(_ context.Context, _, _ string) (string, error) {
	return uuid.NewString(), nil
}
