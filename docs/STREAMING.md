# Streaming Support

Loom supports real-time streaming of LLM responses using Server-Sent Events (SSE).

## Overview

Streaming allows clients to receive AI responses token-by-token as they are generated, providing:
- Faster perceived performance (responses start displaying immediately)
- Better user experience with real-time feedback
- Lower latency for long responses

## Endpoints

### Streaming Chat Completion

**POST** `/api/v1/chat/completions/stream`

Streams chat completion responses in real-time using Server-Sent Events.

**Request Body:**
```json
{
  "provider_id": "provider-123",
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**Response:**

Content-Type: `text/event-stream`

The server sends events in SSE format:

```
event: connected
data: {"message": "Connected to stream"}

event: chunk
data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}

event: chunk
data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" there"}}]}

event: chunk
data: {"id":"1","object":"chat.completion.chunk","created":1234,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}

event: done
data: {"message": "Stream complete"}
```

### Non-Streaming Chat Completion

**POST** `/api/v1/chat/completions`

Standard non-streaming endpoint that returns the complete response at once.

**Query Parameters:**
- `stream=true` - Optional, redirects to streaming endpoint

**Request Body:** Same as streaming endpoint

**Response:**
```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello there! I'm doing well, thank you for asking."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 15,
    "total_tokens": 25
  }
}
```

## Built-in Streaming Test UI

Loom includes a built-in streaming test interface accessible at:

**Web UI → Streaming Test** tab

Features:
- Select any active provider
- Enter test messages
- Toggle between streaming and non-streaming
- Real-time performance metrics (TTFB, chunks/sec, total time)
- Visual streaming indicators with cursor animation
- Compare streaming vs non-streaming performance

## Client Implementation

### JavaScript / Browser (Fetch API with ReadableStream)

```javascript
async function streamChatCompletion(providerId, messages) {
  const response = await fetch('/api/v1/chat/completions/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer YOUR_TOKEN'
    },
    body: JSON.stringify({
      provider_id: providerId,
      messages: messages
    })
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let fullContent = '';

  while (true) {
    const { done, value } = await reader.read();
    
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      if (!line.trim()) continue;
      
      if (line.startsWith('data: ')) {
        const data = line.substring(6);
        
        if (data === '[DONE]') {
          console.log('Stream complete');
          return fullContent;
        }

        try {
          const chunk = JSON.parse(data);
          
          if (chunk.choices && chunk.choices[0]?.delta?.content) {
            const content = chunk.choices[0].delta.content;
            fullContent += content;
            
            // Update UI progressively
            document.getElementById('response').textContent = fullContent;
          }
        } catch (e) {
          // Ignore parse errors
        }
      }
    }
  }

  return fullContent;
}

// Usage
streamChatCompletion('provider-123', [
  {role: 'user', content: 'Hello!'}
]).then(response => {
  console.log('Final response:', response);
}).catch(error => {
  console.error('Stream error:', error);
});
```

### curl

```bash
curl -N -X POST http://localhost:8080/api/v1/chat/completions/stream \
  -H "Content-Type: application/json" \
  -d '{
    "provider_id": "provider-123",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Python

```python
import sseclient
import requests
import json

url = 'http://localhost:8080/api/v1/chat/completions/stream'
data = {
    'provider_id': 'provider-123',
    'messages': [{'role': 'user', 'content': 'Hello!'}]
}

response = requests.post(url, json=data, stream=True)
client = sseclient.SSEClient(response)

for event in client.events():
    if event.event == 'chunk':
        chunk = json.loads(event.data)
        if chunk['choices'] and chunk['choices'][0]['delta'].get('content'):
            print(chunk['choices'][0]['delta']['content'], end='', flush=True)
    elif event.event == 'done':
        print('\n[Stream complete]')
        break
    elif event.event == 'error':
        print(f'\nError: {event.data}')
        break
```

## Provider Compatibility

### Supported Providers

All Loom providers now support streaming:

- **OpenAI-compatible providers** - Full streaming support via SSE format
  - Uses standard `data:` lines with JSON chunks
  - Responds with `[DONE]` marker when complete
  
- **Ollama** - Full streaming support via newline-delimited JSON
  - Uses native Ollama streaming format
  - Automatically converts to OpenAI-compatible chunks
  
- **Mock** - Full streaming support for testing
  - Simulates realistic streaming with configurable delays
  - Useful for development and testing

### Provider Detection

The registry automatically detects streaming support. Requests to providers without streaming will return a 400 error:
```json
{"error": "Provider does not support streaming"}
```

## Error Handling

### Connection Errors
If the provider connection fails, an `error` event is sent:

```
event: error
data: {"error": "Failed to connect to provider"}
```

### Client Disconnection
If the client disconnects, the server automatically cleans up the stream and stops requesting from the provider.

### Timeout
Streaming requests have a 5-minute timeout. For longer responses, the timeout can be adjusted in the code.

## Performance Considerations

### Buffering
Loom disables nginx buffering with `X-Accel-Buffering: no` header to ensure immediate streaming.

### Connection Management
- Each streaming connection maintains an open HTTP connection
- Connections are automatically cleaned up when complete or on error
- Use connection pooling on the client side for multiple requests

### Latency
- Initial response (first token): Typically <500ms
- Subsequent tokens: Near real-time (<50ms between chunks)

## Backwards Compatibility

The non-streaming endpoint `/api/v1/chat/completions` remains fully functional for clients that don't support streaming or prefer complete responses.

## Security

Streaming endpoints respect the same authentication and authorization as other API endpoints. When authentication is enabled, include appropriate headers.

## Implementation Details

### SSE Format
Loom follows the standard SSE specification:
- Each event has an `event:` line specifying the event type
- Each event has a `data:` line with JSON payload
- Events are separated by blank lines

### Chunk Format
Stream chunks follow the OpenAI streaming format:
```json
{
  "id": "unique-id",
  "object": "chat.completion.chunk",
  "created": 1234567890,
  "model": "model-name",
  "choices": [{
    "index": 0,
    "delta": {
      "role": "assistant",  // Only in first chunk
      "content": "text"     // Token content
    },
    "finish_reason": "stop" // Only in last chunk
  }]
}
```

## Provider-Specific Details

### OpenAI Format (SSE)
```
data: {"id":"1","choices":[{"delta":{"content":"Hello"}}]}
data: [DONE]
```

### Ollama Format (newline-delimited JSON)
```json
{"model":"llama2","message":{"content":"Hello"},"done":false}
{"model":"llama2","message":{"content":" there"},"done":true}
```

Both formats are automatically handled and converted to a unified streaming interface.

## Future Enhancements

- [x] Support for Ollama streaming format (✅ Completed)
- [x] Unified streaming interface across providers (✅ Completed)
- [ ] WebSocket alternative to SSE
- [ ] Streaming for function calling
- [ ] Streaming metrics and usage data
- [ ] Rate limiting for streaming requests
