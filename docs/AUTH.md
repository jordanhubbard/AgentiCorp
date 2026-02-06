## API Authentication & Authorization

Loom secures API access with **JWT bearer tokens** and **API keys**. All protected endpoints require one of these:

- **Bearer token:** `Authorization: Bearer <token>` (recommended)
- **API key:** `X-API-Key: <key>`

### Default admin
- Username: `admin`
- Password: `admin`

Change the password immediately:
```
POST /api/v1/auth/change-password
Authorization: Bearer <token>
{
  "current_password": "admin",
  "new_password": "<strong-password>"
}
```

### Login & refresh
```
POST /api/v1/auth/login
{ "username": "admin", "password": "<password>" }

POST /api/v1/auth/refresh
Authorization: Bearer <token>
```

### API keys (service-to-service)
```
POST /api/v1/auth/api-keys
Authorization: Bearer <token>
{ "name": "ci-bot", "permissions": ["agents:read", "beads:read"], "expires_in": 86400 }
```
The full key is returned **once**; store it securely.

### Permissions & roles
- Built-in roles: `admin`, `user`, `viewer`, `service`
- Permissions are resource-scoped (e.g., `agents:read`, `beads:write`) with wildcard support (`*:*`).

### Configuration
`config.yaml` → `security.jwt_secret` (leave empty to auto-generate per run). Set a stable secret in production.

### CORS headers
Authorization headers are allowed: `Access-Control-Allow-Headers: Content-Type, X-API-Key, Authorization`.
