package helpers

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Role mapping to handle different role name formats
// ADMIN role no longer exists - SUPERADMIN is the only admin role
var roleMapping = map[string]string{
	"CUSTOMER SERVICE": "CUSTOMER_SERVICE",
	"CUSTOMER_SERVICE": "CUSTOMER_SERVICE",
	"SUPERADMIN":       "SUPERADMIN", // SUPERADMIN is the only admin role
	"ADMIN":            "SUPERADMIN",
	"NOC":              "NOC",
	"TECHNICIAN":       "TECHNICIAN",
	"FINANCE":          "FINANCE",
	"SIDEKEEPER":       "SIDEKEEPER",
}

func NormalizeRole(role string) string {
	if normalized, exists := roleMapping[role]; exists {
		return normalized
	}
	// Fallback: replace spaces with underscores
	return strings.ReplaceAll(role, " ", "_")
}

func RequireRoles(roles ...string) fiber.Handler {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		// ADMIN role no longer exists - map it to SUPERADMIN for backward compatibility
		if r == "ADMIN" {
			allowed["SUPERADMIN"] = struct{}{}
		} else {
			allowed[r] = struct{}{}
		}
	}
	return func(c *fiber.Ctx) error {
		v := c.Locals("role")
		role, _ := v.(string)

		// Get original role before normalization (for SUPERADMIN check)
		originalRole := role

		// Normalize role name using comprehensive mapping
		normalizedRole := NormalizeRole(role)

		// Debug logging
		slog.Debug("RequireRoles check", "original_role", originalRole, "normalized_role", normalizedRole, "allowed_roles", roles)

		// SUPERADMIN has access to all routes automatically
		if normalizedRole == "SUPERADMIN" || originalRole == "SUPERADMIN" {
			slog.Debug("Access granted to SUPERADMIN (bypasses role check)", "role", originalRole)
			return c.Next()
		}

		// Check if normalized role is allowed
		if _, ok := allowed[normalizedRole]; !ok {
			// Also check original role (for SUPERADMIN which doesn't get normalized)
			if _, ok := allowed[originalRole]; !ok {
				slog.Debug("Access denied", "original_role", originalRole, "normalized_role", normalizedRole)
				return ResponseUtils(c, fiber.StatusForbidden, false, "forbidden", nil)
			}
		}

		slog.Debug("Access granted", "original_role", originalRole, "normalized_role", normalizedRole)
		return c.Next()
	}
}
