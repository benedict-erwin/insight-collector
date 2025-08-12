package auth

import "strings"

// Action constants for permissions
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
	ActionBulk   = "bulk"
	ActionExport = "export"
	ActionAll    = "*"
)

// HasPermission checks if user has required permission with wildcard support
func HasPermission(userPermissions []string, required string) bool {
	for _, perm := range userPermissions {
		if matchesPermission(perm, required) {
			return true
		}
	}
	return false
}

// matchesPermission handles exact match and wildcard patterns
func matchesPermission(userPerm, required string) bool {
	// Exact match
	if userPerm == required {
		return true
	}

	// Super admin access
	if userPerm == "*:*" {
		return true
	}

	// Parse permissions: "action:resource"
	userParts := strings.Split(userPerm, ":")
	requiredParts := strings.Split(required, ":")

	if len(userParts) != 2 || len(requiredParts) != 2 {
		return false
	}

	userAction, userResource := userParts[0], userParts[1]
	reqAction, reqResource := requiredParts[0], requiredParts[1]

	// Admin covers all actions for that resource: "admin:logs" covers "create:logs", "read:logs", etc.
	if userAction == ActionAdmin && userResource == reqResource {
		return true
	}

	// Resource wildcard: "read:*" covers "read:logs", "read:health", etc.
	if userAction == reqAction && userResource == "*" {
		return true
	}

	// Action wildcard: "*:logs" covers "create:logs", "read:logs", etc.
	if userAction == "*" && userResource == reqResource {
		return true
	}

	return false
}