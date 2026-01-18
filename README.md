# arbiter
An agentic based coding orchestrator for both on-prem and off-prem development

## Overview

The Arbiter is an orchestration system that manages interactions between **agents** and **providers**:

- **Agent**: An LLM (Large Language Model) wrapped in glue code that performs tasks
- **Provider**: An AI engine running on-premise or in the cloud (e.g., OpenAI, Anthropic, local models)

The Arbiter maintains its own database for orchestrating all activities and includes a secure key manager for storing provider credentials (API keys) with strong encryption.

## Features

- **Database Management**: SQLite database for storing agents, providers, and orchestration state
- **Secure Key Storage**: Encrypted credential storage using AES-256-GCM with PBKDF2 key derivation
- **Password Protection**: Key store requires password unlock (via environment variable or user prompt)
- **Provider Management**: Register and manage AI providers with optional API credentials
- **Agent Management**: Create and manage agents that use providers
- **Data Isolation**: Arbiter is the sole reader/writer to its database

## Architecture

```
arbiter/
├── cmd/arbiter/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # SQLite database layer (agents & providers)
│   ├── keymanager/      # Encrypted key storage
│   └── models/          # Data models (Agent, Provider)
```

## Installation

```bash
# Build the arbiter
go build -o arbiter ./cmd/arbiter

# Run the arbiter
./arbiter
```

## Usage

### Password Management

The Arbiter requires a password to unlock the key store. You can provide it in two ways:

1. **Environment Variable** (recommended for automation):
```bash
export ARBITER_PASSWORD="your-secure-password"
./arbiter
```

2. **Interactive Prompt** (if environment variable is not set):
```bash
./arbiter
# You will be prompted: "Enter password to unlock key store:"
```

**Security Note**: The password is never stored anywhere and exists only in memory while the Arbiter is running.

### Data Storage

By default, the Arbiter stores data in:
- Database: `~/.local/share/arbiter/arbiter.db`
- Key Store: `~/.local/share/arbiter/keystore.json`

The key store file has restrictive permissions (0600) and contains encrypted API keys.

## Development

### Running Tests

```bash
# Test key manager
go test -v ./internal/keymanager

# Test database
go test -v ./internal/database

# Run all tests
go test -v ./...
```

### Key Manager

The key manager provides secure storage for provider credentials:

- **Encryption**: AES-256-GCM with PBKDF2 key derivation (100,000 iterations)
- **Password-based**: Derived from user password, not stored anywhere
- **Per-key encryption**: Each key has its own salt and nonce
- **Memory safety**: Password cleared from memory when locked

### Database Schema

**Providers Table**:
- Stores AI provider information (OpenAI, Anthropic, local models, etc.)
- Links to encrypted API keys in key manager
- Tracks provider status and configuration

**Agents Table**:
- Stores agent definitions and configurations
- Foreign key relationship to providers
- Tracks agent status and settings

## Security

- **Encrypted Key Storage**: All API keys encrypted at rest using AES-256-GCM
- **No Password Storage**: Unlock password never persisted to disk
- **File Permissions**: Key store created with 0600 permissions (owner read/write only)
- **Key Derivation**: PBKDF2 with SHA-256 and 100,000 iterations
- **Per-key Encryption**: Each key uses unique salt and nonce

## Example

```go
// Create arbiter instance (prompts for password if not in environment)
cfg, _ := config.Default()
arbiter, _ := NewArbiter(cfg)
defer arbiter.Close()

// Register a provider with API key
provider := &models.Provider{
    ID:          "openai-gpt4",
    Name:        "OpenAI GPT-4",
    Type:        "openai",
    Endpoint:    "https://api.openai.com/v1",
    RequiresKey: true,
    Status:      "active",
}
arbiter.CreateProvider(provider, "sk-...your-api-key...")

// Create an agent using the provider
agent := &models.Agent{
    ID:         "coding-agent",
    Name:       "Coding Assistant",
    ProviderID: "openai-gpt4",
    Status:     "active",
    Config:     `{"model": "gpt-4", "temperature": 0.7}`,
}
arbiter.CreateAgent(agent)

// Retrieve agent with provider credentials
agent, provider, apiKey, _ := arbiter.GetAgentWithProvider("coding-agent")
// Use agent, provider, and apiKey to make AI requests...
```

## License

See [LICENSE](LICENSE) file for details.

