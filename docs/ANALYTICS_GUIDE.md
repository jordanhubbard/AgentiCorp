# Analytics & Cost Tracking Guide

Complete guide to using Loom's analytics, cost tracking, and alerting features.

## Table of Contents

- [Overview](#overview)
- [Analytics Dashboard](#analytics-dashboard)
- [Cost Tracking](#cost-tracking)
- [Data Export](#data-export)
- [Alerting System](#alerting-system)
- [Privacy & Security](#privacy--security)
- [API Reference](#api-reference)

## Overview

Loom provides comprehensive analytics to help you:
- **Monitor Usage**: Track requests by provider, model, and user
- **Control Costs**: Real-time cost tracking with budget alerts
- **Optimize Performance**: Monitor latency and error rates
- **Export Data**: Download usage data for external analysis
- **Prevent Overruns**: Automatic alerts for unusual spending

All features are privacy-first with configurable data retention and GDPR compliance.

## Analytics Dashboard

### Accessing the Dashboard

1. Navigate to **Analytics** tab in the UI
2. Or visit: `http://localhost:8080/#analytics`
3. Requires authentication (login first)

### Dashboard Features

#### Summary Cards

Four key metrics displayed at the top:

- **Total Requests**: Number of API calls in selected period
- **Total Cost**: Spending in USD
- **Avg Latency**: Average response time in milliseconds
- **Error Rate**: Percentage of failed requests

#### Time Range Selection

Choose from preset ranges:
- **Last Hour**: Real-time monitoring
- **Last 24 Hours**: Daily overview (default)
- **Last 7 Days**: Weekly trends
- **Last 30 Days**: Monthly analysis
- **Custom Range**: Specify exact dates

#### Visualizations

**Cost Breakdown Charts:**
- Cost by Provider (bar chart)
- Cost by User (bar chart)

**Usage Statistics:**
- Requests by Provider (bar chart)
- Requests by User (bar chart)

**Detailed Table:**
- Provider/User breakdown
- Request counts
- Token usage
- Cost per entity
- Average latency

### Using the Dashboard

#### View Daily Spending

```
1. Select "Last 24 Hours"
2. Click "Refresh"
3. View summary cards for totals
4. Check "Cost by Provider" chart
```

#### Compare Provider Costs

```
1. Select appropriate time range
2. View "Cost by Provider" chart
3. Bars show relative costs
4. Click provider for details (coming soon)
```

#### Monitor User Usage

```
1. Admin only: View all users
2. Regular users: See own usage
3. "Cost by User" shows breakdown
4. Table shows detailed metrics
```

## Cost Tracking

### How Cost Tracking Works

Loom tracks costs based on:
- **Token Usage**: Prompt + completion tokens
- **Provider Pricing**: Cost per million tokens
- **Request Metadata**: User, provider, model

**Formula:**
```
Cost = (Total Tokens / 1,000,000) × Cost Per M Tokens
```

### Viewing Costs

#### Per-Provider Costs

```bash
# Via API
curl -X GET "http://localhost:8080/api/v1/analytics/costs" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response includes:
- Total cost by provider
- Requests per provider
- Cost per request
- Cost per 1K tokens

#### Per-User Costs

Admin can view all users:
```bash
curl -X GET "http://localhost:8080/api/v1/analytics/costs?user_id=user-alice" \
  -H "Authorization: Bearer ADMIN_TOKEN"
```

Regular users see only their own costs automatically.

### Cost Optimization

**Tips to reduce costs:**

1. **Monitor Provider Performance**
   - Check "Cost by Provider" regularly
   - Identify expensive providers
   - Consider alternatives

2. **Track Per-User Spending**
   - Set user budgets
   - Enable alerts (see below)
   - Review usage patterns

3. **Optimize Token Usage**
   - Monitor "Total Tokens" metric
   - Use context efficiently
   - Implement caching (coming soon)

## Data Export

### Export Formats

Loom supports:
- **CSV**: Excel/Sheets compatible
- **JSON**: Programmatic processing

### Export Types

#### 1. Statistics Export

Aggregate data with summaries:

```bash
# JSON format
curl -X GET "http://localhost:8080/api/v1/analytics/export-stats?format=json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o stats.json

# CSV format
curl -X GET "http://localhost:8080/api/v1/analytics/export-stats?format=csv" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o stats.csv
```

**Contains:**
- Summary metrics (requests, tokens, cost, latency, errors)
- Cost breakdown by provider
- Cost breakdown by user
- Request counts per entity

#### 2. Logs Export

Individual request logs:

```bash
# CSV format (recommended)
curl -X GET "http://localhost:8080/api/v1/analytics/export?format=csv" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o logs.csv

# JSON format
curl -X GET "http://localhost:8080/api/v1/analytics/export?format=json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o logs.json
```

**Contains:**
- Timestamp
- User ID
- Provider & Model
- Token counts
- Latency
- Cost
- Status code
- Error messages

### UI Export

Click export buttons in dashboard:
- **Export Stats (JSON)**: Aggregate statistics
- **Export Stats (CSV)**: Aggregate in Excel format
- **Export Logs (CSV)**: Individual requests

### Filtering Exports

Add query parameters:

```bash
# Last 7 days
?start_time=2026-01-14T00:00:00Z&end_time=2026-01-21T00:00:00Z

# Specific provider
?provider_id=provider-openai

# Combined
?start_time=2026-01-20T00:00:00Z&provider_id=provider-openai&format=csv
```

### Scheduled Exports

Automate exports with cron:

```bash
# Daily export at midnight
0 0 * * * curl -X GET "http://localhost:8080/api/v1/analytics/export-stats?format=csv" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o /backups/analytics-$(date +\%Y-\%m-\%d).csv
```

### Excel Integration

1. Export as CSV
2. Open in Excel
3. Data → From Text/CSV
4. Set proper column types
5. Create charts and pivot tables

## Alerting System

### Alert Types

#### 1. Daily Budget Alert

Triggers when daily spending exceeds threshold.

**Default:** $100/day  
**Severity:** Warning  
**Example:** "Daily budget exceeded: $125.00 / $100.00 (125%)"

#### 2. Monthly Budget Alert

Triggers when monthly spending exceeds threshold.

**Default:** $2000/month  
**Severity:** Critical  
**Example:** "Monthly budget exceeded: $2350.00 / $2000.00 (117%)"

#### 3. Anomaly Detection

Triggers when spending is unusually high compared to history.

**Default:** 2x normal spending  
**Severity:** Warning  
**Example:** "Unusual spending detected: $50.00 today vs $20.00 average (2.5x increase)"

### Configuring Alerts

**AlertConfig structure:**

```go
type AlertConfig struct {
    UserID              string  // User to monitor
    DailyBudgetUSD      float64 // Daily threshold ($100 default)
    MonthlyBudgetUSD    float64 // Monthly threshold ($2000 default)
    AnomalyThreshold    float64 // Multiplier for anomaly (2.0 = 2x)
    EnableEmailAlerts   bool    // Send email notifications
    EnableWebhookAlerts bool    // Send webhook notifications
    WebhookURL          string  // Webhook endpoint
    EmailAddress        string  // Email recipient
}
```

**Example Configuration:**

```go
config := &analytics.AlertConfig{
    UserID:           "user-alice",
    DailyBudgetUSD:   50.0,  // $50/day limit
    MonthlyBudgetUSD: 1000.0, // $1000/month limit
    AnomalyThreshold: 3.0,   // Alert if 3x normal
    EnableEmailAlerts: true,
    EmailAddress:     "alice@company.com",
}

checker := analytics.NewAlertChecker(storage, config)
alerts, err := checker.CheckAlerts(ctx)
```

### Alert Notifications

**Current:** Logs to console  
**Coming Soon:**
- Email notifications via SMTP
- Webhook POST requests
- Slack/Teams integrations
- SMS via Twilio

**Webhook Payload Example:**

```json
{
  "alert_id": "alert-daily-1234567890",
  "user_id": "user-alice",
  "type": "budget_exceeded",
  "severity": "warning",
  "message": "Daily budget exceeded: $125.00 / $100.00 (125%)",
  "current_cost": 125.00,
  "threshold": 100.00,
  "triggered_at": "2026-01-21T14:30:00Z"
}
```

### Best Practices

1. **Set Realistic Budgets**
   - Analyze 30 days of history
   - Add 20% buffer for growth
   - Update quarterly

2. **Tune Anomaly Threshold**
   - Start with 2x multiplier
   - Reduce if too many false positives
   - Increase for more tolerance

3. **Test Notifications**
   - Trigger test alert
   - Verify delivery
   - Check spam folders

4. **Monitor Alerts**
   - Review triggered alerts weekly
   - Acknowledge resolved issues
   - Adjust thresholds as needed

## Privacy & Security

### Data Logged

**By Default (Privacy-First):**
- ✅ Request metadata (method, path, timestamp)
- ✅ Token counts and costs
- ✅ Latency and status codes
- ❌ Request bodies (redacted)
- ❌ Response bodies (redacted)

### Privacy Controls

**Configure logging:**

```go
privacy := &analytics.PrivacyConfig{
    LogRequestBodies:  false, // Don't log request bodies
    LogResponseBodies: false, // Don't log response bodies
    RedactPatterns: []string{
        // Email addresses
        `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
        // API keys
        `(?i)(api[_-]?key|token|secret)["\s:=]+([a-zA-Z0-9_-]{20,})`,
    },
    MaxBodyLength: 10000, // 10KB max if bodies enabled
}

logger := analytics.NewLogger(storage, privacy)
```

### PII Redaction

Automatically redacts:
- Email addresses
- API keys and tokens
- Credit card numbers
- Social security numbers
- Custom patterns

### GDPR Compliance

- **Right to Access**: Export API provides user data
- **Right to Deletion**: Purge logs before retention date
- **Data Minimization**: Privacy-first defaults
- **Purpose Limitation**: Logs used only for analytics
- **Storage Limitation**: Configurable retention

**Purge old logs:**

```go
// Delete logs older than 90 days
ninetyDaysAgo := time.Now().Add(-90 * 24 * time.Hour)
deleted, err := logger.PurgeLogs(ctx, ninetyDaysAgo)
```

### Access Control

- **Regular Users**: See only their own data
- **Admin Users**: Can filter by any user ID
- **API Keys**: Require appropriate permissions
- **JWT Tokens**: Must be valid and not expired

## API Reference

For complete API documentation, see [ANALYTICS_API.md](ANALYTICS_API.md).

### Quick Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/logs` | GET | Retrieve request logs |
| `/api/v1/analytics/stats` | GET | Get aggregate statistics |
| `/api/v1/analytics/costs` | GET | Get cost breakdown |
| `/api/v1/analytics/export` | GET | Export logs (CSV/JSON) |
| `/api/v1/analytics/export-stats` | GET | Export stats (CSV/JSON) |

**Authentication:** All endpoints require `Authorization: Bearer <token>`

## Troubleshooting

### No Data in Dashboard

**Check:**
1. Time range includes actual usage
2. Authentication token is valid
3. User has made requests
4. Database is accessible

### Incorrect Cost Calculations

**Verify:**
1. Provider pricing is configured
2. Token counts are accurate
3. No duplicate logs
4. Currency conversion (if applicable)

### Alerts Not Triggering

**Debug:**
1. Check AlertConfig thresholds
2. Verify user ID matches
3. Review historical spending
4. Check for errors in logs

### Export Fails

**Try:**
1. Reduce time range
2. Check disk space
3. Verify authentication
4. Use smaller batch sizes

## Examples

### Monitor Daily Spending

```bash
# Get today's costs
curl -X GET "http://localhost:8080/api/v1/analytics/costs?start_time=$(date -u +%Y-%m-%dT00:00:00Z)" \
  -H "Authorization: Bearer YOUR_TOKEN" | jq .
```

### Find Most Expensive Provider

```bash
# Export stats and process
curl -X GET "http://localhost:8080/api/v1/analytics/export-stats?format=json" \
  -H "Authorization: Bearer YOUR_TOKEN" | \
  jq '.cost_by_provider | to_entries | sort_by(.value) | reverse | .[0]'
```

### Weekly Cost Report

```bash
#!/bin/bash
# weekly_report.sh
WEEK_AGO=$(date -u -d '7 days ago' +%Y-%m-%dT00:00:00Z)
NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)

curl -X GET "http://localhost:8080/api/v1/analytics/costs?start_time=$WEEK_AGO&end_time=$NOW" \
  -H "Authorization: Bearer $TOKEN" | \
  jq '{
    total_cost: .total_cost_usd,
    total_requests: .total_requests,
    avg_cost_per_request: .cost_per_request,
    top_provider: (.cost_by_provider | to_entries | max_by(.value) | .key)
  }'
```

## See Also

- [Analytics API Reference](ANALYTICS_API.md)
- [Architecture Documentation](ARCHITECTURE.md)
- [User Guide](USER_GUIDE.md)
- [Agent Development](../AGENTS.md)
