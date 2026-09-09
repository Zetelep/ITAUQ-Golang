package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SupabasePasswordUpdater updates an Auth user's password using Supabase's
// admin API (service role key required). Used by the change-password flow so
// the user can rotate the temporary password issued at provisioning time.
type SupabasePasswordUpdater struct {
	URL            string
	ServiceRoleKey string
	Client         *http.Client
}

func (u *SupabasePasswordUpdater) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	payload, _ := json.Marshal(map[string]string{"password": newPassword})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		strings.TrimRight(u.URL, "/")+"/auth/v1/admin/users/"+userID,
		bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", u.ServiceRoleKey)
	req.Header.Set("Authorization", "Bearer "+u.ServiceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	client := u.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("update auth user password: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("update auth user password: supabase returned %s: %s", resp.Status, string(body))
	}
	return nil
}

// MemoryPasswordUpdater is the dev/in-memory fallback so the usecase remains
// functional when DATABASE_URL / Supabase credentials are not configured.
type MemoryPasswordUpdater struct{}

func (MemoryPasswordUpdater) UpdatePassword(_ context.Context, _ string, _ string) error {
	return nil
}
