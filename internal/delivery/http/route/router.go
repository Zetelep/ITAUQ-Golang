package route

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	appUsecase "github.com/itauq-golang/internal/usecase/application"
	elUsecase "github.com/itauq-golang/internal/usecase/evaluationlink"
	evalUsecase "github.com/itauq-golang/internal/usecase/evaluation"
	eligUsecase "github.com/itauq-golang/internal/usecase/eligibility"
	pfUsecase "github.com/itauq-golang/internal/usecase/publicflow"
	profileUsecase "github.com/itauq-golang/internal/usecase/profile"
	qUsecase "github.com/itauq-golang/internal/usecase/questionnaire"
	tsUsecase "github.com/itauq-golang/internal/usecase/taskscenario"
	adminUsecase "github.com/itauq-golang/internal/usecase/administrator"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
	"github.com/jackc/pgx/v5/pgxpool"
)

// db is set by Setup; when nil the middleware falls back to the previous
// header-based behavior (useful for in-memory/dev mode).
var db *pgxpool.Pool

// extractSubFromJWT decodes the JWT payload and returns the "sub" claim
// without verifying the signature. TODO: replace with proper signature
// verification (JWKS/HMAC) for production.
func extractSubFromJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", errors.New("invalid jwt")
	}
	payload := parts[1]
	// Try standard RawURLEncoding then fall back to URLEncoding
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			return "", err
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(b, &claims); err != nil {
		return "", err
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("sub claim not found")
	}
	return sub, nil
}

// extractUserContext extracts user ID and role from JWT and stores them in gin context.
// This middleware should be used after authentication to make user info available to handlers.
func extractUserContext(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.Next()
		return
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		c.Next()
		return
	}
	token := parts[1]

	if db == nil {
		// Fallback: use headers in dev mode
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			c.Set("user_id", userID)
		}
		if userRole := c.GetHeader("X-User-Role"); userRole != "" {
			c.Set("user_role", userRole)
		}
		c.Next()
		return
	}

	sub, err := extractSubFromJWT(token)
	if err != nil {
		c.Next()
		return
	}
	c.Set("user_id", sub)

	var role string
	row := db.QueryRow(c, `SELECT roles FROM public.profiles WHERE id=$1`, sub)
	if err := row.Scan(&role); err == nil {
		c.Set("user_role", role)
	}
	c.Next()
}

// RequireAdmin allows requests from users whose profile role is "administrator"
// or "super_admin". When db == nil it falls back to checking the
// X-User-Role header (previous behavior).
func RequireAdmin(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "authorization required"}})
		return
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "invalid authorization header"}})
		return
	}
	token := parts[1]
	if db == nil {
		// Fallback: trust X-User-Role header (development mode)
		if c.GetHeader("X-User-Role") != "administrator" && c.GetHeader("X-User-Role") != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "admin role required"}})
			return
		}
		c.Next()
		return
	}
	// Production path: extract sub from JWT and look up role in profiles
	sub, err := extractSubFromJWT(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token"}})
		return
	}
	var role string
	row := db.QueryRow(c, `SELECT roles FROM public.profiles WHERE id=$1`, sub)
	if err := row.Scan(&role); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "admin role required"}})
		return
	}
	if role != "administrator" && role != "super_admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "admin role required"}})
		return
	}
	c.Next()
}

// RequireSuperAdmin allows only users whose profile role is "super_admin".
// Falls back to X-User-Role header if db == nil.
func RequireSuperAdmin(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "authorization required"}})
		return
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "invalid authorization header"}})
		return
	}
	token := parts[1]
	if db == nil {
		// Fallback: previous header-based behavior
		if c.GetHeader("X-User-Role") != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "super admin role required"}})
			return
		}
		c.Next()
		return
	}
	// Production path: extract sub from JWT and look up role in profiles
	sub, err := extractSubFromJWT(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token"}})
		return
	}
	var role string
	row := db.QueryRow(c, `SELECT roles FROM public.profiles WHERE id=$1`, sub)
	if err := row.Scan(&role); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "super admin role required"}})
		return
	}
	if role != "super_admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "super admin role required"}})
		return
	}
	c.Next()
}

// Setup registers routes and stores the DB pool for middleware use.
// Public, token-gated respondent endpoints are mounted *before* the auth
// middleware so they never see a JWT — they're keyed by the URL token alone.
func Setup(r *gin.Engine, appUC *appUsecase.Usecase, qUC *qUsecase.Usecase, tsUC *tsUsecase.Usecase, elUC *elUsecase.Usecase, pfUC *pfUsecase.Usecase, profileUC *profileUsecase.Usecase, evalUC *evalUsecase.Usecase, adminUC *adminUsecase.Usecase, eligUC *eligUsecase.Usecase, instrument *itauq.Loaded, susInstrument *sus.Loaded, d *pgxpool.Pool) {
	db = d
	SetupPublicFlow(r, pfUC)
	r.Use(extractUserContext)
	SetupApplications(r, appUC, RequireSuperAdmin)
	SetupAdministrators(r, adminUC, RequireSuperAdmin)
	SetupQuestionnaires(r, qUC, RequireAdmin)
	SetupTaskScenarios(r, tsUC, RequireAdmin)
	SetupEligibility(r, eligUC, RequireAdmin)
	SetupEvaluationLinks(r, elUC, RequireAdmin)
	SetupProfile(r, profileUC, RequireAdmin)
	SetupInstruments(r, instrument, susInstrument, RequireAdmin)
	SetupEvaluation(r, evalUC, RequireAdmin)
}
