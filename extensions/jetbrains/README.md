# Loom JetBrains Plugin

AI-powered coding assistant for IntelliJ IDEA, PyCharm, WebStorm, and all JetBrains IDEs.

## Features

- **AI Chat Tool Window**: Persistent chat panel with conversation history
- **Code Actions**: Right-click context menu for code assistance
- **Inline Suggestions**: AI-powered code completions
- **Multi-IDE Support**: Works with all JetBrains IDEs

## Installation

### From Marketplace

1. Open IDE Settings: `File` → `Settings` (Windows/Linux) or `IntelliJ IDEA` → `Preferences` (macOS)
2. Select `Plugins`
3. Click `Marketplace` tab
4. Search for "Loom"
5. Click `Install`
6. Restart IDE

### From ZIP

1. Download plugin ZIP from releases
2. `Settings` → `Plugins` → ⚙️ → `Install Plugin from Disk...`
3. Select downloaded ZIP file
4. Restart IDE

## Configuration

1. `Settings` → `Tools` → `Loom`
2. Configure:
   - **API Endpoint**: Loom server URL (default: `http://localhost:8080`)
   - **API Key**: Optional authentication key
   - **Model**: Preferred AI model
   - **Enable Inline Suggestions**: Toggle code completions

## Usage

### Chat Window

1. Click Loom icon in right sidebar
2. Type question in input field
3. Press `Ctrl+Enter` to send

### Code Actions

1. Select code
2. Right-click → `Loom`
3. Choose action:
   - Explain Code
   - Generate Tests
   - Refactor
   - Help Fix Bug

### Inline Suggestions

- Start typing code
- AI suggestions appear inline (grey text)
- Press `Tab` to accept
- Press `Esc` to dismiss

## Building

```bash
cd extensions/jetbrains
./gradlew buildPlugin
```

Plugin JAR created in `build/distributions/`

## Development

```bash
# Run IDE with plugin
./gradlew runIde

# Run tests
./gradlew test

# Verify plugin
./gradlew verifyPlugin
```

## Requirements

- JetBrains IDE 2023.2 or later
- Java 17 or later
- Loom server running

## Supported IDEs

- IntelliJ IDEA
- PyCharm
- WebStorm
- PhpStorm
- GoLand
- RubyMine
- CLion
- DataGrip
- Rider
- Android Studio

## Architecture

```
LoomClient.kt       - API client
ChatToolWindowFactory.kt  - Chat UI
LoomCompletionContributor.kt - Inline completions
Actions/*.kt              - Context menu actions
Settings/*.kt             - Configuration UI
```

## License

MIT
