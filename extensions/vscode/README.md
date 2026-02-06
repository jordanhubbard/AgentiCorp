# Loom VS Code Extension

Integrate Loom's AI-powered coding assistant directly into Visual Studio Code.

## Features

### 💬 AI Chat Panel

- Persistent chat interface in VS Code sidebar
- Context-aware conversations
- Code-aware responses
- Conversation history

### 📝 Code Actions

Right-click on any selected code to:
- **Ask About Selection**: Ask custom questions about selected code
- **Explain Code**: Get detailed explanations
- **Generate Tests**: Create comprehensive unit tests
- **Refactor Code**: Improve code quality and readability
- **Fix Bugs**: Get help debugging issues

### ⚙️ Configuration

Configure the extension via VS Code settings:

```json
{
  "loom.apiEndpoint": "http://localhost:8080",
  "loom.apiKey": "",
  "loom.model": "default",
  "loom.autoContext": true,
  "loom.maxContextLines": 100
}
```

## Requirements

- Loom server running (default: `http://localhost:8080`)
- VS Code 1.85.0 or higher

## Installation

### From VSIX

1. Download the latest `.vsix` file from releases
2. Open VS Code
3. Go to Extensions (`Ctrl+Shift+X`)
4. Click `...` menu → "Install from VSIX..."
5. Select the downloaded file

### From Source

```bash
cd extensions/vscode
npm install
npm run compile
npm run package  # Creates .vsix file
```

## Quick Start

1. **Start Loom server**:
   ```bash
   docker compose up -d
   # Or: ./loom
   ```

2. **Open chat panel**:
   - Click Loom icon in Activity Bar
   - Or: `Ctrl+Shift+P` → "Loom: Open Chat"

3. **Ask a question**:
   - Type in chat panel
   - Or select code → Right-click → Loom actions

## Usage Examples

### Chat Panel

```
You: How do I read a file in Go?
Loom: Here's how to read a file in Go...
```

### Code Explanation

1. Select code
2. Right-click → "Explain this code"
3. View explanation in chat panel

### Generate Tests

1. Select function/method
2. Right-click → "Generate tests"
3. Tests appear in chat
4. Copy and paste into test file

### Bug Fixing

1. Select buggy code
2. Right-click → "Help fix this bug"
3. Describe the issue
4. Get suggestions

## Keyboard Shortcuts

- `Ctrl+Shift+P` → "Loom: Open Chat" - Open chat panel
- Right-click selection → Loom menu - Code actions
- `Ctrl+Enter` in chat - Send message

## Configuration Options

### `loom.apiEndpoint`

Loom API endpoint URL.

- Type: `string`
- Default: `"http://localhost:8080"`

### `loom.apiKey`

Optional API key for authentication.

- Type: `string`
- Default: `""`

### `loom.model`

Preferred AI model to use.

- Type: `string`
- Default: `"default"`

### `loom.autoContext`

Automatically include file context in requests.

- Type: `boolean`
- Default: `true`

### `loom.maxContextLines`

Maximum lines of context to include.

- Type: `number`
- Default: `100`

## Troubleshooting

### Connection Error

**Problem**: "Loom API is not reachable"

**Solution**:
1. Verify Loom server is running: `curl http://localhost:8080/health`
2. Check `loom.apiEndpoint` setting
3. Check firewall/network settings

### Authentication Error

**Problem**: 401 Unauthorized

**Solution**:
1. Set `loom.apiKey` in settings
2. Verify API key is valid
3. Check server authentication configuration

### Slow Responses

**Problem**: Responses take too long

**Solution**:
1. Check server load: `curl http://localhost:8080/health`
2. Reduce `loom.maxContextLines`
3. Use faster model in settings

## Development

### Setup

```bash
git clone https://github.com/jordanhubbard/Loom.git
cd Loom/extensions/vscode
npm install
```

### Build

```bash
npm run compile
```

### Watch Mode

```bash
npm run watch
```

### Package

```bash
npm run package
```

Creates `loom-vscode-1.0.0.vsix`

### Test

```bash
npm run test
```

### Debug

1. Open in VS Code
2. Press `F5` to launch Extension Development Host
3. Test extension in new window

## Architecture

```
extension.ts       - Extension activation and commands
client.ts          - Loom API client
chatPanel.ts       - Chat UI webview provider
```

### API Integration

The extension communicates with Loom via REST API:

```typescript
POST /api/v1/chat/completions
{
  "messages": [
    {"role": "user", "content": "..."}
  ],
  "model": "default"
}
```

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Commit changes: `git commit -m "Add feature"`
4. Push to branch: `git push origin feature/my-feature`
5. Open pull request

## Support

- **Documentation**: https://github.com/jordanhubbard/Loom
- **Issues**: https://github.com/jordanhubbard/Loom/issues
- **Discussions**: https://github.com/jordanhubbard/Loom/discussions

## License

MIT License - see LICENSE file

## Changelog

See [CHANGELOG.md](CHANGELOG.md)

---

**Powered by Loom** 🤖
