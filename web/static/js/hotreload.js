// Hot-Reload Client
// Connects to WebSocket and reloads page when files change

(function() {
    // Only enable in development
    const isDevelopment = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';

    if (!isDevelopment) {
        console.log('[HotReload] Disabled (not in development environment)');
        return;
    }

    let ws = null;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;
    const reconnectDelay = 2000;

    function connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/hotreload`;

        console.log('[HotReload] Connecting to:', wsUrl);

        ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            console.log('[HotReload] Connected');
            reconnectAttempts = 0;
            showNotification('Hot-reload enabled', 'info');
        };

        ws.onmessage = function(event) {
            try {
                const data = JSON.parse(event.data);
                handleMessage(data);
            } catch (err) {
                console.error('[HotReload] Failed to parse message:', err);
            }
        };

        ws.onerror = function(error) {
            console.error('[HotReload] WebSocket error:', error);
        };

        ws.onclose = function() {
            console.log('[HotReload] Disconnected');
            ws = null;

            // Attempt to reconnect
            if (reconnectAttempts < maxReconnectAttempts) {
                reconnectAttempts++;
                console.log(`[HotReload] Reconnecting in ${reconnectDelay}ms (attempt ${reconnectAttempts}/${maxReconnectAttempts})`);
                setTimeout(connect, reconnectDelay);
            } else {
                console.log('[HotReload] Max reconnect attempts reached');
                showNotification('Hot-reload disconnected', 'warning');
            }
        };
    }

    function handleMessage(data) {
        console.log('[HotReload] Message:', data);

        switch (data.type) {
            case 'connected':
                // Initial connection confirmed
                break;

            case 'file_changed':
                handleFileChange(data);
                break;

            default:
                console.log('[HotReload] Unknown message type:', data.type);
        }
    }

    function handleFileChange(data) {
        const path = data.path;
        const operation = data.operation;

        console.log(`[HotReload] File ${operation}: ${path}`);

        // Determine file type
        const ext = path.split('.').pop().toLowerCase();

        switch (ext) {
            case 'html':
                reloadPage('HTML file changed');
                break;

            case 'css':
                reloadCSS();
                break;

            case 'js':
                reloadPage('JavaScript file changed');
                break;

            case 'json':
            case 'yaml':
            case 'yml':
                // Only reload for config files in web/static, not beads data
                if (path.includes('/static/') || path === 'config.yaml') {
                    reloadPage('Config file changed');
                } else {
                    console.log(`[HotReload] Ignoring non-static data file: ${path}`);
                }
                break;

            default:
                console.log(`[HotReload] Ignoring change to ${ext} file`);
        }
    }

    function reloadCSS() {
        console.log('[HotReload] Reloading CSS...');

        const links = document.querySelectorAll('link[rel="stylesheet"]');
        links.forEach(link => {
            const href = link.getAttribute('href');
            if (href) {
                // Add cache-busting timestamp
                const newHref = href.split('?')[0] + '?t=' + Date.now();
                link.setAttribute('href', newHref);
            }
        });

        showNotification('CSS reloaded', 'success');
    }

    function reloadPage(reason) {
        // Defer reload while a modal is open to avoid losing user edits
        if (window.modalState && window.modalState.activeId) {
            console.log(`[HotReload] Deferring reload (modal open): ${reason}`);
            return;
        }
        console.log(`[HotReload] Reloading page: ${reason}`);
        showNotification(`Reloading: ${reason}`, 'info');

        // Small delay to show notification
        setTimeout(() => {
            window.location.reload();
        }, 500);
    }

    function showNotification(message, type) {
        // Use existing toast notification system if available
        if (window.showToast) {
            const toastType = type === 'success' ? 'success' :
                             type === 'warning' ? 'warning' :
                             type === 'error' ? 'error' : 'info';
            window.showToast(message, toastType);
            return;
        }

        // Fallback: console log
        console.log(`[HotReload] ${type.toUpperCase()}: ${message}`);
    }

    // Start connection
    connect();

    // Expose API for debugging
    window.hotReload = {
        reconnect: connect,
        status: function() {
            return {
                connected: ws && ws.readyState === WebSocket.OPEN,
                attempts: reconnectAttempts
            };
        },
        test: function() {
            reloadPage('Manual test');
        }
    };

    console.log('[HotReload] Client initialized. Use window.hotReload for debugging.');
})();
