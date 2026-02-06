# Hot-Reload System

## Overview

The hot-reload system enables rapid development by automatically detecting file changes and notifying connected browsers to reload. This eliminates the need to manually refresh the browser after editing frontend code.

## Architecture

### Components

1. **File Watcher** (`internal/hotreload/watcher.go`)
   - Uses `fsnotify` to monitor file system changes
   - Watches specified directories and file patterns
   - Debounces rapid changes (e.g., multiple saves)
   - Notifies subscribers of file changes

2. **WebSocket Server** (`internal/hotreload/server.go`)
   - Manages WebSocket connections from browsers
   - Broadcasts file change events to all connected clients
   - Handles client connection/disconnection

3. **Manager** (`internal/hotreload/manager.go`)
   - Coordinates watcher and WebSocket server
   - Configurable via `config.yaml`

4. **Client** (`web/static/js/hotreload.js`)
   - Connects to WebSocket endpoint
   - Listens for file change events
   - Reloads CSS without full page refresh
   - Triggers full page reload for JS/HTML changes

## Configuration

### config.yaml

```yaml
hot_reload:
  enabled: true  # Enable hot-reload (development only)
  watch_dirs:
    - "./web/static"  # Watch frontend files
    - "./personas"    # Watch persona definitions
  patterns:
    - "*.html"  # HTML files
    - "*.css"   # Stylesheets
    - "*.js"    # JavaScript
    - "*.md"    # Markdown (personas)
    - "PERSONA.md"
```

### Settings

- **enabled**: Turn hot-reload on/off (disable in production)
- **watch_dirs**: List of directories to monitor
- **patterns**: File patterns to watch (glob format)

## Integration (To Complete)

### Step 1: Initialize in main.go

```go
import (
	"github.com/jordanhubbard/loom/internal/hotreload"
)

func main() {
	// ... existing config loading ...

	// Initialize hot-reload
	var hrManager *hotreload.Manager
	if cfg.HotReload.Enabled {
		var err error
		hrManager, err = hotreload.NewManager(
			cfg.HotReload.Enabled,
			cfg.HotReload.WatchDirs,
			cfg.HotReload.Patterns,
		)
		if err != nil {
			log.Printf("Hot-reload initialization failed: %v", err)
		} else {
			defer hrManager.Close()
		}
	}

	// ... rest of initialization ...
}
```

### Step 2: Register WebSocket Route

```go
// In main.go after apiServer.SetupRoutes()
handler := apiServer.SetupRoutes()

// Add hot-reload WebSocket endpoint
if hrManager != nil && hrManager.IsEnabled() {
	http.HandleFunc("/ws/hotreload", hrManager.GetServer().HandleWebSocket)
	http.HandleFunc("/api/v1/hotreload/status", hrManager.GetServer().HandleStatus)
	log.Println("[HotReload] WebSocket endpoint registered at /ws/hotreload")
}
```

### Step 3: Include Client Script

Add to `web/static/index.html` before closing `</body>` tag:

```html
<!-- Hot-reload client (development only) -->
<script src="/static/js/hotreload.js"></script>
```

## Workflow

```
┌─────────────────────┐
│  Developer Edits    │
│  File (app.js)      │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  fsnotify Detects   │
│  Change Event       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Watcher Debounces  │
│  (100ms delay)      │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Watcher Notifies   │
│  WebSocket Server   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Server Broadcasts  │
│  to All Clients     │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Browser Receives   │
│  Message            │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Client Reloads     │
│  ├─ CSS: Hot reload │
│  └─ JS/HTML: Full   │
└─────────────────────┘
```

## Client Behavior

### CSS Changes
- Reload stylesheets without full page refresh
- Add cache-busting timestamp to `<link>` tags
- Page state preserved

### JavaScript Changes
- Full page reload required
- Show notification before reload
- 500ms delay to show message

### HTML Changes
- Full page reload required
- Immediate reload

### Configuration Changes
- Full page reload
- Useful for persona definition updates

## Browser Compatibility

- Requires WebSocket support
- Modern browsers (Chrome, Firefox, Safari, Edge)
- Falls back gracefully if WebSocket unavailable

## Development Usage

### Enable Hot-Reload

```bash
# In config.yaml
hot_reload:
  enabled: true
```

### Check Status

```bash
curl http://localhost:8080/api/v1/hotreload/status
```

Response:
```json
{
  "enabled": true,
  "connected_clients": 2
}
```

### Debug in Browser Console

```javascript
// Check connection status
window.hotReload.status()
// { connected: true, attempts: 0 }

// Manual reload test
window.hotReload.test()

// Reconnect manually
window.hotReload.reconnect()
```

## Production Deployment

**IMPORTANT**: Disable hot-reload in production:

```yaml
hot_reload:
  enabled: false
```

Hot-reload introduces:
- Additional file I/O overhead
- WebSocket connections
- Potential security considerations

It's designed for development environments only.

## Troubleshooting

### Hot-Reload Not Working

1. **Check configuration**
   ```bash
   grep -A 10 "hot_reload" config.yaml
   ```

2. **Verify WebSocket connection**
   - Open browser DevTools → Network → WS tab
   - Should see connection to `ws://localhost:8080/ws/hotreload`
   - Status should be "101 Switching Protocols"

3. **Check server logs**
   ```bash
   grep "HotReload" logs/loom.log
   ```

   Should see:
   ```
   [HotReload] Watching ./web/static
   [HotReload] Client connected (total: 1)
   ```

### Changes Not Detected

1. **Verify watch directories exist**
   ```bash
   ls -la ./web/static
   ```

2. **Check file patterns match**
   - Patterns use filepath.Match (not regex)
   - Example: `*.js` matches `app.js` but not `js/app.js`
   - Use glob patterns: `**/*.js` for recursive

3. **Look for watcher errors**
   ```bash
   grep "Watcher error" logs/loom.log
   ```

### Browser Not Reloading

1. **Check hotreload.js is loaded**
   - Open DevTools → Sources
   - Verify `/static/js/hotreload.js` is present

2. **Check for console errors**
   ```javascript
   // Should see:
   [HotReload] Connected
   [HotReload] Client initialized
   ```

3. **Test WebSocket manually**
   ```javascript
   const ws = new WebSocket('ws://localhost:8080/ws/hotreload');
   ws.onopen = () => console.log('Connected');
   ws.onmessage = (e) => console.log('Message:', e.data);
   ```

### Too Many Reconnect Attempts

If hot-reload repeatedly tries to reconnect:
- Server might not be running
- WebSocket endpoint not registered
- Check max reconnect attempts (default: 5)

## Performance Considerations

### File System Overhead

- **Debouncing**: Changes are debounced (100ms) to avoid excessive events
- **Recursive watching**: Automatically watches subdirectories
- **Pattern filtering**: Only processes matching file types

### Memory Usage

- Each connected browser = 1 WebSocket connection
- Minimal memory per connection (~10KB)
- Event buffer size: 10 events per client

### Recommended Limits

- **Watch directories**: 1-5 directories
- **File patterns**: 3-10 patterns
- **Connected clients**: 1-10 browsers

For large codebases (>10,000 files), consider:
- Limiting watch patterns to specific extensions
- Watching only actively developed directories

## Future Enhancements

### Phase 1 (Current)
- ✅ File watching with fsnotify
- ✅ WebSocket server for notifications
- ✅ Browser client with hot CSS reload
- ✅ Configuration support

### Phase 2 (Planned)
- ⏳ Hot module replacement for JavaScript
- ⏳ Persona reload without page refresh
- ⏳ Backend code hot-reload (partial - Go)
- ⏳ Selective component refresh

### Phase 3 (Future)
- 📋 Browser sync across multiple devices
- 📋 File change history/timeline
- 📋 Conditional reload (e.g., only if no console errors)
- 📋 Integration with test runner

## See Also

- [Auto-Bug Dispatch](./auto-bug-dispatch.md)
- [Agent Bug Fix Workflow](./agent-bug-fix-workflow.md)
- [Development Setup](../QUICKSTART.md)
