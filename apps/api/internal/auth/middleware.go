package auth

// Service satisfies middleware.AuthService via its ValidateAPIKey(ctx, rawKey) method
// defined in service.go. No adapter is needed because the interface was written to
// accept context.Context, not *fiber.Ctx, keeping this package free of Fiber imports.
