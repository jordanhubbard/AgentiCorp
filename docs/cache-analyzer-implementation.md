# Cache Opportunities Analyzer Implementation

## Overview

The Cache Opportunities Analyzer analyzes request patterns to identify caching opportunities that can reduce costs and improve performance. It examines historical request logs, detects duplicate requests, calculates potential savings, and provides actionable recommendations.

## Architecture

### Components

1. **Analyzer** (`internal/cache/analyzer.go`)
   - Main analysis engine
   - Duplicate detection using SHA256 hashing
   - Savings calculation
   - TTL recommendation
   - Priority assignment

2. **Types** (`internal/cache/types_analyzer.go`)
   - `DuplicateRequest` - Detected duplicate pattern
   - `CacheOpportunity` - Actionable caching recommendation
   - `AnalysisReport` - Comprehensive analysis results
   - `AnalysisConfig` - Analysis configuration
   - `OptimizationResult` - Results of applying optimizations

3. **API Handlers** (`internal/api/handlers_cache_analyzer.go`)
   - `GET /api/v1/cache/analysis` - Full analysis report
   - `GET /api/v1/cache/opportunities` - List opportunities
   - `POST /api/v1/cache/optimize` - Apply optimizations
   - `GET /api/v1/cache/recommendations` - Get recommendations

## How It Works

### 1. Duplicate Detection

The analyzer:
1. Fetches request logs within the configured time window (default: 7 days)
2. Hashes each request using `SHA256(provider:model:request_body)`
3. Groups requests by hash to identify duplicates
4. Filters requests that occur less than `MinOccurrences` times (default: 2)

**Example Hash Calculation:**
```go
func hashRequest(providerID, modelName, requestBody string) string {
    hasher := sha256.New()
    hasher.Write([]byte(providerID + ":" + modelName + ":" + requestBody))
    return hex.EncodeToString(hasher.Sum(nil))
}
```

### 2. Savings Calculation

For each duplicate pattern:

**Potential Cache Hits** = `Total Occurrences - 1` (first request still needs to hit the provider)

**Token Savings** = `Average Tokens * Potential Hits`

**Cost Savings** = `Average Cost * Potential Hits`

**Latency Savings** = `Average Latency * Potential Hits`

**Hit Rate** = `Potential Hits / Total Occurrences * 100`

### 3. TTL Recommendation

The analyzer suggests TTL based on request timing:

1. Calculate average interval between duplicate requests
2. Multiply by 2x for safety margin
3. Clamp to reasonable bounds (5 minutes to 24 hours)
4. Round to nearest hour (or 15 minutes for sub-hour TTLs)

**Example:**
- 10 requests over 20 minutes
- Average interval: 2.2 minutes
- Suggested TTL: 2.2 * 2 = 4.4 minutes → rounded to 5 minutes

### 4. Priority Assignment

**High Priority:**
- Cost savings > $1.00
- Hit rate > 70%

**Medium Priority:**
- Cost savings > $0.10
- Hit rate > 50%

**Low Priority:**
- Everything else that meets minimum thresholds

### 5. Auto-Enableability

Opportunities are marked auto-enableable if:
- `AutoEnable` config is true
- Monthly projected savings >= `AutoEnableMinUSD` (default: $10/month)
- Hit rate >= `AutoEnableMinRate` (default: 50%)

## Configuration

### Default Configuration

```go
config := cache.DefaultAnalysisConfig()
// {
//     TimeWindow:        7 * 24 * time.Hour,  // 7 days
//     MinOccurrences:    2,                    // At least 2 duplicates
//     MinSavingsUSD:     0.01,                 // At least $0.01 savings
//     AutoEnable:        false,                // Don't auto-enable
//     AutoEnableMinUSD:  10.0,                 // $10/month for auto-enable
//     AutoEnableMinRate: 0.5,                  // 50% hit rate for auto-enable
// }
```

### Custom Configuration

```go
config := &cache.AnalysisConfig{
    TimeWindow:        24 * time.Hour,  // Last 24 hours
    MinOccurrences:    3,                // At least 3 duplicates
    MinSavingsUSD:     0.50,             // At least $0.50 savings
    AutoEnable:        true,             // Enable auto-optimization
    AutoEnableMinUSD:  5.0,              // $5/month threshold
    AutoEnableMinRate: 0.6,              // 60% hit rate threshold
}
```

## API Usage

### 1. Run Full Analysis

```bash
curl http://localhost:8080/api/v1/cache/analysis
```

**Optional Query Parameters:**
- `time_window` - Duration string (e.g., "24h", "7d")
- `min_savings` - Minimum savings in USD (e.g., "0.50")
- `auto_enable` - Boolean to include auto-enable recommendations

**Response:**
```json
{
  "analyzed_at": "2026-01-31T23:57:52Z",
  "time_window": 604800000000000,
  "total_requests": 15,
  "unique_requests": 2,
  "duplicate_count": 13,
  "duplicate_percent": 86.67,
  "opportunities": [
    {
      "id": "e022b382-866f-4f56-8f32-adb2403a34cf",
      "pattern": "anthropic:claude-sonnet-4-5",
      "description": "Duplicate requests to anthropic using claude-sonnet-4-5",
      "request_count": 10,
      "potential_hits": 9,
      "hit_rate_percent": 90,
      "tokens_savable": 9000,
      "cost_savable_usd": 0.135,
      "latency_savable_ms": 18000,
      "recommendation": "Strongly recommend enabling caching for anthropic:claude-sonnet-4-5 (90% hit rate, $0.14 savings, 5m TTL)",
      "auto_enableable": false,
      "suggested_ttl": 300000000000,
      "priority": "medium",
      "provider_id": "anthropic",
      "model_name": "claude-sonnet-4-5"
    }
  ],
  "total_savings_usd": 0.183,
  "total_tokens_saved": 12200,
  "total_latency_saved_ms": 24000,
  "monthly_projection_usd": 0.78,
  "recommendations": [
    "Enable caching to save approximately $0.78 per month",
    "Recommended TTL: 5m0s for most patterns"
  ]
}
```

### 2. List Opportunities

```bash
curl http://localhost:8080/api/v1/cache/opportunities
```

**Optional Query Parameters:**
- `limit` - Max opportunities to return (default: 20)
- `priority` - Filter by priority: "high", "medium", "low"

**Response:**
```json
{
  "opportunities": [ /* opportunity objects */ ],
  "count": 2,
  "total": 2,
  "limit": 20
}
```

### 3. Get Recommendations

```bash
curl http://localhost:8080/api/v1/cache/recommendations
```

**Response:**
```json
{
  "recommendations": [
    "Enable caching to save approximately $0.78 per month",
    "Priority: Enable caching for anthropic:claude-sonnet-4-5 (90% hit rate, $0.14 savings)",
    "Recommended TTL: 5m for most patterns",
    "2 opportunities qualify for auto-optimization"
  ],
  "total_savings_usd": 0.183,
  "monthly_projection": 0.78,
  "duplicate_percent": 86.67,
  "opportunity_count": 2,
  "analyzed_at": "2026-01-31T23:57:52Z",
  "time_window": "168h0m0s"
}
```

### 4. Apply Optimizations

```bash
curl -X POST http://localhost:8080/api/v1/cache/optimize \
  -H "Content-Type: application/json" \
  -d '{
    "opportunity_ids": ["e022b382-866f-4f56-8f32-adb2403a34cf"],
    "auto_enable": false
  }'
```

**Request Body:**
- `opportunity_ids` - Array of opportunity IDs to enable (optional)
- `auto_enable` - If true and no IDs provided, enables all auto-enableable opportunities

**Response:**
```json
{
  "applied_count": 1,
  "skipped_count": 0,
  "total_savings_usd": 0.135,
  "applied_patterns": ["anthropic:claude-sonnet-4-5"],
  "skipped_patterns": [],
  "errors": []
}
```

## Implementation Details

### Database Integration

The analyzer uses the analytics storage layer:

```go
// Get database
db := loom.GetDatabase()

// Create analytics storage
storage, err := analytics.NewDatabaseStorage(db.DB())

// Create analyzer
analyzer := cache.NewAnalyzer(storage, config)

// Run analysis
report, err := analyzer.Analyze(ctx)
```

### Request Log Requirements

The analyzer expects request logs with:
- `ProviderID` - Provider identifier
- `ModelName` - Model name
- `RequestBody` - Full request body (for hashing)
- `TotalTokens` - Token count
- `CostUSD` - Request cost
- `LatencyMs` - Request latency
- `StatusCode` - HTTP status (only analyzes 200-399)
- `Timestamp` - Request timestamp

### Filtering Logic

Requests are excluded from analysis if:
- Status code >= 400 (failed requests)
- Occurrence count < `MinOccurrences`
- Cost savings < `MinSavingsUSD`
- Outside the configured time window

## Example Workflow

1. **Monitor requests** for 7 days (default time window)
2. **Run analysis** to identify duplicate patterns
3. **Review opportunities** sorted by savings potential
4. **Apply optimizations** for high-value patterns
5. **Measure results** by comparing before/after metrics
6. **Adjust TTLs** based on cache hit rates
7. **Re-analyze** periodically to find new opportunities

## Best Practices

### Analysis Frequency

- Run analysis weekly for stable workloads
- Run daily for rapidly changing request patterns
- Run after major feature launches or traffic changes

### Interpreting Results

- **High duplicate %** (>80%): Strong caching potential
- **Medium duplicate %** (50-80%): Selective caching recommended
- **Low duplicate %** (<50%): Evaluate cost/benefit carefully

### TTL Tuning

- Start with suggested TTLs
- Monitor cache hit rates
- Increase TTL if hit rate is high and content is stable
- Decrease TTL if getting stale responses

### Auto-Enablement

- Only enable for patterns with consistent savings >$10/month
- Ensure hit rates >50% before auto-enabling
- Review auto-enabled patterns weekly

## Integration with Caching System

**Note:** The `/api/v1/cache/optimize` endpoint currently returns a simulation of what would be enabled. Full integration with the caching system requires:

1. Updating cache configuration in database
2. Enabling semantic caching for provider/model combinations
3. Setting TTLs for cache entries
4. Logging optimization actions in activity feed

## Monitoring and Metrics

Track these metrics to measure analyzer effectiveness:

- **Duplicate Detection Rate**: % of duplicates found
- **Savings Accuracy**: Actual vs projected savings
- **TTL Effectiveness**: Cache hit rates by pattern
- **Auto-Enable Success**: % of auto-enabled patterns that deliver ROI

## Future Enhancements

- **Pattern-based caching**: Detect similar (not identical) requests
- **Semantic similarity**: Use embeddings to find cacheable variations
- **Dynamic TTL adjustment**: Automatically tune TTLs based on hit rates
- **Cost alerts**: Notify when high-value opportunities are detected
- **A/B testing**: Compare caching strategies
- **Multi-provider optimization**: Recommend cheaper providers for cacheable requests

## Related Components

- **Semantic Caching** (`internal/cache/semantic.go`) - Prompt caching implementation
- **Analytics Storage** (`internal/analytics/database.go`) - Request log persistence
- **Activity Feed** - Logs optimization actions
- **Notifications** - Alerts for high-value opportunities

## Troubleshooting

### No opportunities detected

- Verify request logs are being saved
- Check time window covers sufficient data
- Lower `MinOccurrences` or `MinSavingsUSD` thresholds
- Ensure requests have `TotalTokens` and `CostUSD` populated

### Low hit rates

- Review duplicate detection logic
- Check if request bodies vary slightly (parameters, timestamps)
- Consider semantic similarity caching
- Verify cache is properly enabled

### Inaccurate savings projections

- Ensure cost data is accurate
- Verify token counts are correct
- Check if traffic patterns are seasonal
- Compare actual savings after enabling caching

## References

- Bead: `ac-8j2` (Cache Opportunities Analyzer)
- Related beads:
  - `ac-p3z` (Semantic Caching)
  - `ac-g1k` (Analytics Dashboard)
