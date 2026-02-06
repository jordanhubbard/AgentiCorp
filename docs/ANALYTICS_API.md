# Analytics & Export API Documentation

This document describes the analytics and data export API endpoints in Loom.

## Overview

The Analytics API provides endpoints for viewing usage statistics, tracking costs, and exporting data for external analysis.

## Authentication

All analytics endpoints require authentication via JWT token or API key:

```http
Authorization: Bearer <token>
```

## Endpoints

### Get Request Logs

Retrieve individual request logs with filtering.

```http
GET /api/v1/analytics/logs
```

**Query Parameters:**
- `provider_id` (optional): Filter by provider ID
- `start_time` (optional): Start time in RFC3339 format
- `end_time` (optional): End time in RFC3339 format
- `limit` (optional): Maximum number of results (default: 100)
- `offset` (optional): Offset for pagination

**Response:**
```json
[
  {
    "id": "log-123",
    "timestamp": "2026-01-21T12:00:00Z",
    "user_id": "user-alice",
    "method": "POST",
    "path": "/api/v1/chat/completions",
    "provider_id": "provider-openai",
    "model_name": "gpt-4",
    "prompt_tokens": 100,
    "completion_tokens": 50,
    "total_tokens": 150,
    "latency_ms": 1200,
    "status_code": 200,
    "cost_usd": 0.003
  }
]
```

### Get Analytics Statistics

Get aggregate statistics for usage, costs, and performance.

```http
GET /api/v1/analytics/stats
```

**Query Parameters:**
- `user_id` (optional, admin only): Filter by user ID
- `start_time` (optional): Start time in RFC3339 format
- `end_time` (optional): End time in RFC3339 format

**Response:**
```json
{
  "total_requests": 1250,
  "total_tokens": 150000,
  "total_cost_usd": 4.5,
  "avg_latency_ms": 950.5,
  "error_rate": 0.02,
  "requests_by_user": {
    "user-alice": 800,
    "user-bob": 450
  },
  "requests_by_provider": {
    "provider-openai": 900,
    "provider-anthropic": 350
  },
  "cost_by_provider": {
    "provider-openai": 3.2,
    "provider-anthropic": 1.3
  },
  "cost_by_user": {
    "user-alice": 2.8,
    "user-bob": 1.7
  }
}
```

### Get Cost Report

Get detailed cost breakdown and metrics.

```http
GET /api/v1/analytics/costs
```

**Query Parameters:**
- `user_id` (optional, admin only): Filter by user ID
- `start_time` (optional): Start time in RFC3339 format
- `end_time` (optional): End time in RFC3339 format

**Response:**
```json
{
  "total_cost_usd": 4.5,
  "total_requests": 1250,
  "total_tokens": 150000,
  "cost_per_request": 0.0036,
  "cost_per_1k_tokens": 0.03,
  "cost_by_provider": {
    "provider-openai": 3.2,
    "provider-anthropic": 1.3
  },
  "cost_by_user": {
    "user-alice": 2.8,
    "user-bob": 1.7
  },
  "time_range": {
    "start": "2026-01-20T00:00:00Z",
    "end": "2026-01-21T00:00:00Z"
  }
}
```

### Export Request Logs

Export individual request logs in CSV or JSON format.

```http
GET /api/v1/analytics/export?format=csv
GET /api/v1/analytics/export?format=json
```

**Query Parameters:**
- `format` (required): Export format (`csv` or `json`)
- `provider_id` (optional): Filter by provider ID
- `start_time` (optional): Start time in RFC3339 format
- `end_time` (optional): End time in RFC3339 format

**CSV Format:**
```csv
Timestamp,User ID,Method,Path,Provider ID,Model,Prompt Tokens,Completion Tokens,Total Tokens,Latency (ms),Status Code,Cost (USD),Error Message
2026-01-21T12:00:00Z,user-alice,POST,/api/v1/chat/completions,provider-openai,gpt-4,100,50,150,1200,200,0.0030,
```

**Response:**
- Content-Type: `text/csv` or `application/json`
- Content-Disposition: `attachment; filename="loom-logs-YYYY-MM-DD.{csv,json}"`

### Export Statistics Summary

Export aggregate statistics in CSV or JSON format.

```http
GET /api/v1/analytics/export-stats?format=csv
GET /api/v1/analytics/export-stats?format=json
```

**Query Parameters:**
- `format` (required): Export format (`csv` or `json`)
- `user_id` (optional, admin only): Filter by user ID
- `start_time` (optional): Start time in RFC3339 format
- `end_time` (optional): End time in RFC3339 format

**JSON Format:**
```json
{
  "exported_at": "2026-01-21T14:30:00Z",
  "time_range": {
    "start": "2026-01-20T00:00:00Z",
    "end": "2026-01-21T00:00:00Z"
  },
  "summary": {
    "total_requests": 1250,
    "total_tokens": 150000,
    "total_cost_usd": 4.5,
    "avg_latency_ms": 950.5,
    "error_rate": 0.02
  },
  "cost_by_provider": { ... },
  "cost_by_user": { ... },
  "requests_by_provider": { ... },
  "requests_by_user": { ... }
}
```

**CSV Format:**
```csv
Summary,,,
Metric,Value,,
Total Requests,1250,,
Total Tokens,150000,,
Total Cost (USD),4.5000,,
Avg Latency (ms),950.50,,
Error Rate,2.00%,,

Cost by Provider,,,
Provider ID,Requests,Cost (USD),
provider-openai,900,3.2000,
provider-anthropic,350,1.3000,

Cost by User,,,
User ID,Requests,Cost (USD),
user-alice,800,2.8000,
user-bob,450,1.7000,
```

## Usage Examples

### Export Last 7 Days (CSV)

```bash
curl -X GET "https://api.loom.example/api/v1/analytics/export-stats?format=csv&start_time=2026-01-14T00:00:00Z&end_time=2026-01-21T00:00:00Z" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o analytics-export.csv
```

### Get Cost Breakdown for Specific User (Admin)

```bash
curl -X GET "https://api.loom.example/api/v1/analytics/costs?user_id=user-alice" \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

### Export Full Request Logs (JSON)

```bash
curl -X GET "https://api.loom.example/api/v1/analytics/export?format=json&start_time=2026-01-21T00:00:00Z" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o logs-export.json
```

## Rate Limits

- Export endpoints: 10 requests per minute per user
- Query endpoints: 60 requests per minute per user
- Large exports (>10,000 records) may be throttled

## Privacy & Security

- Users can only access their own data by default
- Admins can access all users' data via `user_id` parameter
- Request/response bodies are NOT included in exports by default
- PII is automatically redacted from logs
- All exports are logged for audit purposes

## Excel Integration

CSV exports can be opened directly in Microsoft Excel, Google Sheets, or LibreOffice Calc. For advanced Excel features:

1. Export data as CSV
2. Open in Excel
3. Use Excel's "Data" → "From Text/CSV" to import with proper formatting
4. Save as `.xlsx` if needed

## Scheduled Exports

For automated exports, use cron or a scheduler to call the export API:

```bash
# Daily export at midnight (crontab example)
0 0 * * * curl -X GET "https://api.loom.example/api/v1/analytics/export-stats?format=csv" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o /backups/analytics-$(date +\%Y-\%m-\%d).csv
```

## See Also

- [Analytics Dashboard](USER_GUIDE.md#analytics-dashboard)
- [Privacy Configuration](ARCHITECTURE.md#privacy-controls)
- [Cost Tracking](../AGENTS.md#cost-tracking)
