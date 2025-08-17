# Insight Collector

Go HTTP service for logging and time-series data management. Built for high-performance data ingestion with smart pagination, background job processing, and real-time monitoring.

## Quick Start

```bash
# Start services with InfluxDB v2 OSS (default/production)
docker-compose up -d

# OR for research/experimental with v3 Core
docker-compose -f docker-compose-v3.yml up -d

# Enter container
docker exec -it insight-collector /bin/ash

# Start HTTP server with hot reload (using air)
air -c .air.toml

# Start worker
go run main.go worker start
```

## Architecture

```
├── cmd/           # CLI commands
├── config/        # Configuration management
├── http/
│   ├── middleware/    # Custom middleware
│   ├── v1/handler/    # API handlers v1
│   ├── v1/route/      # Route definitions v1
│   ├── v2/handler/    # API handlers v2
│   ├── v2/route/      # Route definitions v2
│   └── registry/      # Route registry
├── internal/
│   ├── entities/      # Data structures
│   ├── jobs/          # Background jobs
│   └── services/      # Business logic
└── pkg/
    ├── asynq/         # Queue management
    ├── influxdb/      # Database client with cursor-based pagination
    ├── logger/        # Structured logging
    ├── maxmind/       # GeoIP with auto-downloader
    ├── redis/         # Centralized Redis client
    └── utils/         # Utilities
```

## Adding New API Feature

### 1. Create Entity (if needed)
```bash
# File: internal/entities/user.go
type User struct {
    ID    string `json:"id" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}
```

### 2. Create Service
```bash
# File: internal/services/user.go
func CreateUser(user entities.User) error {
    // Business logic here
    return nil
}
```

### 3. Create Handler
```bash
# File: http/v1/handler/user.go
func CreateUser(c echo.Context) error {
    var user entities.User
    if err := c.Bind(&user); err != nil {
        return err
    }
    
    // Validate request
    if err := c.Validate(&user); err != nil {
        return err
    }
    
    result, err := services.CreateUser(user)
    if err != nil {
        return response.Error(c, err.Error(), 400)
    }
    
    return response.Success(c, result)
}
```

### 4. Create Route
```bash
# File: http/v1/route/user.go
func init() {
    registry.Register("v1", func(g *echo.Group) {
        g.POST("/users", handler.CreateUser)
    })
}
```

## Adding New Background Job

### 1. Single Registration Point (Type-Safe)
```bash
# File: internal/jobs/registry.go
import "your-app/internal/constants"

func RegisterHandlers(mux *asynq.ServeMux) ([]JobRegistration, error) {
    jobs := []JobRegistration{
        // Existing jobs...
        {
            TaskType: ua.TypeUserActivitiesLogging,
            Handler:  ua.HandleUserActivitiesLogging,
            Queue:    constants.QueueCritical, // Type-safe constant
        },
        // NEW JOB - just add this entry!
        {
            TaskType: "notifications:send",
            Handler:  notifications.HandleNotificationSend,
            Queue:    constants.QueueDefault, // Will get 30% allocation
        },
    }
    
    // Queue validation ensures only valid constants are used
    return jobs, nil
}
```

### 2. Queue Constants (Available Options)
```bash
# File: internal/constants/queues.go
const (
    QueueCritical = "critical" // 60% - High priority (user activities, security, payments)
    QueueDefault  = "default"  // 30% - Normal priority (notifications, processing)
    QueueLow      = "low"      // 10% - Background (cleanup, reports, analytics)
)
```

### 3. Create Job Handler
```bash
# File: internal/jobs/notifications/notifications.go
const TypeNotificationSend = "notifications:send"

func HandleNotificationSend(ctx context.Context, t *asynq.Task) error {
    // Get task metadata from Asynq
    taskID := t.ResultWriter().TaskID()
    taskType := t.Type()
    
    // Unmarshal business data only
    var notification entities.Notification
    if err := json.Unmarshal(t.Payload(), &notification); err != nil {
        return err
    }
    
    // Scoped logging with task correlation
    log := logger.WithScope("notification-job")
    log.Info().Str("task_id", taskID).Str("task_type", taskType).Msg("Processing")
    
    // Send notification logic here
    return sendNotification(notification)
}
```

### 4. Dispatch Job (Structured Payload)
```bash
# In your handler (e.g., http/v1/handler/notifications.go)
func NotificationHandler(c echo.Context) error {
    jobID := fmt.Sprintf("notification_%d_%s", utils.Now().Unix(), constants.GetRequestID(c)[:8])
    
    // Use structured payload
    err := asynq.DispatchJob(&asynq.Payload{
        TaskID:   jobID,
        TaskType: notifications.TypeNotificationSend, // Type constant
        Data:     notificationData,
    })
    
    if err != nil {
        return response.Fail(c, http.StatusInternalServerError, 1, "Failed to dispatch job")
    }
    
    return response.Success(c, map[string]interface{}{
        "message": "Notification queued",
        "job_id":  jobID,
    })
}
```

### 5. Workers Auto-Generated!
```bash
# Current: 2 queues with jobs
# critical: 60% (user_activities:logging)
# low: 40% (example:processing)

# After adding job to QueueDefault - 3 queues:
# critical: 60% (user_activities:logging) 
# default: 30% (notifications:send)         ← NEW! Gets 30%
# low: 10% (example:processing)             ← Reduced to intended 10%

# No manual configuration needed - workers auto-adjust!
# Optional: Customize allocation if needed
./app worker set default 50 notifications:send,other:tasks
```

## Worker Management

Enhanced CLI-based worker management with JSON output and incremental task addition:

```bash
# List workers in clean JSON format (no table truncation)
./app worker list
# Output: {"concurrency":10,"workers":[{"queue":"critical","percentage":86,"task_types":[...]}]}

# Show specific worker details in JSON
./app worker show critical
# Output: {"queue":"critical","percentage":86,"count":8,"task_types":[...]}

# Set complete worker configuration (replaces all task types)
./app worker set critical 70 user_activities:logging,security_events:logging

# ✨ NEW: Add task types incrementally (preserves existing ones)
./app worker add critical new_feature:logging,custom_task:processing
# Automatically prevents duplicates, maintains existing configuration

# Worker lifecycle management
./app worker start        # Start background worker
./app worker status       # Show queue weights and active configuration
./app worker concurrency 20  # Update worker count (requires restart)
./app worker validate     # Check configuration validity
./app worker reset        # Reset to auto-generated from job registry

# JSON output can be processed with jq for automation
./app worker list 2>/dev/null | jq -r '.workers[] | "\(.queue): \(.task_types | length) tasks"'
```

### Worker Management Best Practices

```bash
# Development workflow:
1. ./app worker list                    # Check current config
2. ./app worker add critical new_task:logging   # Add new task incrementally
3. ./app worker validate               # Ensure configuration is valid
4. pkill -f "worker start" && ./app worker start &  # Restart worker

# Production deployment:
1. ./app worker reset                  # Clean slate from job registry
2. ./app worker add critical prod_feature:logging  # Add production tasks
3. ./app worker concurrency 50        # Scale for production load
4. ./app worker validate              # Final validation
```

## InfluxDB Integration

### Production Setup (InfluxDB v2 OSS - Recommended)

For **production environments**, use InfluxDB v2 OSS with the provided Docker Compose setup:

```bash
# Services available:
# - InfluxDB v2: http://localhost:8086 (admin/password123)
# - Grafana: http://localhost:3000 (admin/admin)
```

**Configuration for v2 OSS:**
```json
{
  "influxdb": {
    "version": "v2-oss",
    "url": "http://localhost:8086",
    "token": "your-influxdb-token",
    "org": "insight",
    "bucket": "insight-logs"
  }
}
```

**Query Language - Flux:**
```flux
// Get recent user activities
from(bucket: "insight-logs")
  |> range(start: -24h)
  |> filter(fn: (r) => r["_measurement"] == "user_activities")
  |> filter(fn: (r) => r["status"] == "success")

// Aggregation by currency
from(bucket: "insight-logs")
  |> range(start: -1h)
  |> filter(fn: (r) => r["_measurement"] == "transaction_events")
  |> group(columns: ["currency"])
  |> sum(column: "_value")
```

### Research/Experimental (InfluxDB v3 Core)

⚠️ **Note**: InfluxDB v3 Core support is provided for **research and exploration purposes only**. Do not use in production environments.

```bash
# V3 is for experimental use only
```

**Configuration for v3 Core (experimental):**
```json
{
  "influxdb": {
    "version": "v3-core",
    "host": "influxdb3-core",
    "port": 8181,
    "token": "your-token",
    "auth_scheme": "bearer",
    "bucket": "insight-logs"
  }
}
```

**Query Language - SQL (experimental):**
```sql
-- SQL syntax for v3 Core
SELECT * FROM user_activities 
WHERE time >= now() - INTERVAL '24 hours'
  AND status = 'success'

-- Aggregation
SELECT currency, SUM(amount) as total
FROM transaction_events
WHERE time >= now() - INTERVAL '1 hour'
GROUP BY currency
```

### Usage (No Code Changes Required)

The wrapper system automatically handles version differences. Your existing code works exactly the same:

```go
// Same initialization
err := influxdb.Init()
if err != nil {
    log.Fatal(err)
}
defer influxdb.Close()

// Same entity usage
userActivity := &useractivities.UserActivities{
    UserID: "user123",
    ActivityType: "login",
    Timestamp: time.Now(),
}

// Same function calls
point := userActivity.ToPoint()
err = influxdb.WritePoint(point)

// Same querying (but different query syntax per version)
iterator, err := influxdb.Query("your-flux-or-sql-query")
if err != nil {
    return err
}
defer iterator.Close()

for iterator.Next() {
    record := iterator.Record()
    fmt.Printf("Record: %+v\n", record)
}
```

## InfluxDB Cursor-Based Pagination System

### Overview
Production-ready cursor-based pagination system for InfluxDB v2 OSS time-series data, optimized for large datasets (100k+ records) with true server-side efficiency.

### Key Features
- **🚀 True Server-Side Pagination**: RFC3339 timestamp cursors prevent network overhead
- **🎯 Dynamic Filtering**: Configurable tag/field validation with security controls
- **📅 Date Range Support**: Exact dates and ranges (YYYY-MM-DD format)
- **🛡️ Safety Limits**: 10x multiplier (50-1000 cap) for first page loads
- **📊 Accurate Counting**: Entity-specific CountField for unique record totals
- **⚡ Memory Efficient**: Automatic cleanup of internal InfluxDB fields

### Quick Usage

#### 1. Entity Configuration
```go
// File: internal/entities/user_activities/query_config.go
func GetQueryConfig(bucket string) v2oss.QueryBuilderConfig {
    return v2oss.QueryBuilderConfig{
        Bucket:      bucket,
        Measurement: "user_activities",
        ValidTags:   map[string]bool{"user_id": true, "status": true},
        ValidFields: map[string]bool{"duration_ms": true, "response_code": true},
        CountField:  "request_id", // For unique counting
    }
}
```

#### 2. Handler Implementation  
```go
// File: http/v1/handler/user_activities.go
func ListHandler(c echo.Context) error {
    var req v2oss.PaginationRequest
    if err := c.Bind(&req); err != nil {
        return response.FailWithCodeAndMessage(c, constants.CodeInvalidJSON, err.Error())
    }
    
    // Create query builder and executor
    cfg := config.Get()
    queryConfig := uaEntities.GetQueryConfig(cfg.InfluxDB.Bucket)
    qb := v2oss.NewQueryBuilder(queryConfig)
    executor := v2oss.NewInfluxDBExecutor()
    
    // Execute queries with safety limits
    totalRecords := qb.GetTotalCount(&req, executor)
    results, err := qb.ExecuteDataQuery(&req, executor)
    if err != nil {
        return response.Fail(c, http.StatusInternalServerError, 3, "Failed to execute data query")
    }
    
    // Build response with cursor metadata
    paginationInfo := qb.GetPaginationInfo(&req, results, totalRecords)
    responseData := v2oss.PaginationResponse{
        Data:       results,
        Pagination: paginationInfo,
    }
    
    return response.Success(c, responseData)
}
```

### API Request/Response

#### Request Format
```bash
curl -X POST http://localhost:8080/v1/user-activities/list \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "length": 25,
    "cursor": "2024-01-15T10:30:00Z",
    "direction": "next",
    "filters": [
      {"key": "status", "value": "success"},
      {"key": "user_id", "value": "user123"}
    ],
    "range": {
      "start": "2024-01-15",
      "end": "2024-01-16"
    }
  }'
```

#### Response Format
```json
{
  "success": true,
  "code": 0,
  "data": {
    "data": [
      {
        "_time": "2024-01-15T10:29:45Z",
        "user_id": "user123",
        "session_id": "sess456",
        "activity_type": "login",
        "status": "success",
        "duration_ms": 150,
        "request_id": "req789"
      }
    ],
    "pagination": {
      "length": 25,
      "has_next": true,
      "has_prev": true,
      "next_cursor": "2024-01-15T10:25:30Z",
      "prev_cursor": "2024-01-15T10:29:45Z",
      "direction": "next",
      "total": 1250
    }
  },
  "message": "Successful",
  "request_id": "req-1234567890-abcdef12"
}
```

### Performance Optimizations
- **First Page Safety**: 50-1000 record limit (10x requested length) prevents catastrophic data transfer
- **Cursor Efficiency**: Subsequent pages use timestamp filtering for fast navigation  
- **Memory Management**: Automatic cleanup of internal fields (result, table, _start, _stop)
- **Query Timeout**: 60-second timeout with proper context cancellation
- **Connection Pooling**: Direct InfluxDB client integration

### Extending to Other Entities

Add pagination to new entities in 3 steps:

1. **Create query config**:
```go
func GetSecurityEventsQueryConfig(bucket string) v2oss.QueryBuilderConfig {
    return v2oss.QueryBuilderConfig{
        Bucket:      bucket,
        Measurement: "security_events",
        ValidTags:   map[string]bool{"threat_level": true},
        CountField:  "event_id",
    }
}
```

2. **Add handler**: Same pattern as user activities
3. **Register route**: Add to route file with authentication

### Architecture Benefits
- **Import Cycle Free**: Dependency injection pattern avoids circular imports
- **Type Safe**: Interface-based design with proper error handling
- **Reusable**: Query builder works across different entity types
- **Testable**: Mockable interfaces for unit testing
- **Production Ready**: Memory limits, timeouts, and error handling

## MaxMind GeoIP Integration

### Features
- **Thread-safe GeoIP lookups** for City and ASN data
- **Automatic database downloader** with SHA256 verification  
- **Conditional downloads** based on Last-Modified headers
- **Periodic update checking** (configurable interval, default 1 hour)
- **Basic authentication** with MaxMind account credentials
- **Graceful fallback** when databases unavailable
- **CLI management commands** for download control

### Configuration
Add MaxMind configuration to `.config.json`:

```json
{
  "maxmind": {
    "enabled": true,
    "storage_path": "storage/maxmind", 
    "check_interval": "1h",
    "databases": {
      "city": "GeoLite2-City",
      "asn": "GeoLite2-ASN"  
    },
    "downloader": {
      "enabled": true,
      "account_id": "YOUR_ACCOUNT_ID",
      "license_key": "YOUR_LICENSE_KEY",
      "base_url": "https://download.maxmind.com/geoip/databases",
      "timeout": "30s",
      "retry_attempts": 3,
      "retry_delay": "5s"
    },
    "cache": {
      "enabled": true,
      "max_entries": 10000,
      "ttl": "1h"
    }
  }
}
```

### Cache Configuration
- **enabled**: Enable/disable LRU caching for GeoIP lookups (default: `true`)
- **max_entries**: Maximum number of IP addresses to cache (default: `10000`)
- **ttl**: Time-to-live for cached entries (default: `"1h"`)

**Cache Benefits:**
- **Reduced Database I/O**: Repeated IP lookups served from memory
- **Improved Performance**: ~2-5x faster for workloads with repeated IP addresses
- **Memory Efficient**: LRU eviction prevents unbounded memory growth
- **Thread-Safe**: Concurrent access without performance degradation

### CLI Commands

```bash
# Check for database updates
./insight-collector maxmind check-updates

# Force download specific database  
./insight-collector maxmind download city
./insight-collector maxmind download asn

# Force download all databases (both City & ASN)  
./insight-collector maxmind download all

# Show service status
./insight-collector maxmind status
./insight-collector maxmind status --json

# Show detailed database info
./insight-collector maxmind info --json

# Test IP lookup
./insight-collector maxmind lookup 8.8.8.8
./insight-collector maxmind lookup 1.1.1.1 --json
```

### API Usage

```go
// Initialize service
if err := maxmind.Init(); err != nil {
    log.Fatal(err)
}
defer maxmind.Close()

// Perform GeoIP lookups
geoLocation := maxmind.LookupCityFromString("8.8.8.8")
fmt.Printf("Country: %s, City: %s\n", geoLocation.Country, geoLocation.City)

asnInfo := maxmind.LookupASNFromString("8.8.8.8")  
fmt.Printf("ASN: %d, Org: %s\n", asnInfo.ASN, asnInfo.Organization)

// Manual operations
err := maxmind.CheckForUpdates()
err := maxmind.ForceDownload("city")
status := maxmind.GetDownloadStatus()
```

### Environment Variables
Override credentials via environment variables:
- `MAXMIND_ACCOUNT_ID` - MaxMind account ID
- `MAXMIND_LICENSE_KEY` - MaxMind license key

### Download Process
1. **HEAD request** to check Last-Modified header
2. **Download SHA256 checksum** for verification
3. **Download database TAR.GZ** file  
4. **Verify checksum** before extraction
5. **Extract .mmdb file** to storage directory
6. **Automatic service reload** with new database

## Configuration

**File: `.config.json`**
```json
{
  "app": {
    "name": "InsightCollector",
    "port": 8080
  },
  "redis": {
    "mode": "single",
    "host": "redis",
    "port": 6379,
    "password": "",
    "db": 0,
    "cluster": {
      "nodes": ["redis-1:6379", "redis-2:6379", "redis-3:6379"],
      "password": ""
    },
    "pools": {
      "default": {
        "size": 300,
        "timeout": "5s",
        "dial_timeout": "3s",
        "read_timeout": "2s",
        "write_timeout": "2s",
        "max_lifetime": "30m",
        "idle_timeout": "10m"
      },
      "asynq": {
        "size": 200,
        "timeout": "10s",
        "dial_timeout": "5s",
        "read_timeout": "3s",
        "write_timeout": "3s",
        "max_lifetime": "30m",
        "idle_timeout": "5m"
      },
      "sessions": {
        "size": 50,
        "timeout": "3s",
        "dial_timeout": "2s",
        "read_timeout": "1s",
        "write_timeout": "1s",
        "max_lifetime": "15m",
        "idle_timeout": "2m"
      },
      "cache": {
        "size": 100,
        "timeout": "5s",
        "dial_timeout": "3s",
        "read_timeout": "2s",
        "write_timeout": "2s",
        "max_lifetime": "20m",
        "idle_timeout": "5m"
      },
      "nonce": {
        "size": 20,
        "timeout": "2s",
        "dial_timeout": "1s",
        "read_timeout": "1s",
        "write_timeout": "1s",
        "max_lifetime": "10m",
        "idle_timeout": "1m"
      }
    }
  },
  "asynq": {
    "concurrency": 200,
    "db": 0,
    "pool_size": 200
  },
  "influxdb": {
    "version": "v2-oss",
    "url": "http://localhost:8086",
    "token": "your-influxdb-token",
    "org": "insight",
    "bucket": "insight-logs"
  },
  "maxmind": {
    "enabled": true,
    "storage_path": "storage/maxmind",
    "check_interval": "1h",
    "databases": {
      "city": "GeoLite2-City",
      "asn": "GeoLite2-ASN"
    },
    "downloader": {
      "enabled": true,
      "account_id": "YOUR_ACCOUNT_ID",
      "license_key": "YOUR_LICENSE_KEY",
      "base_url": "https://download.maxmind.com/geoip/databases",
      "timeout": "30s",
      "retry_attempts": 3,
      "retry_delay": "5s"
    },
    "cache": {
      "enabled": true,
      "max_entries": 10000,
      "ttl": "1h"
    }
  },
  "auth": {
    "enabled": true,
    "algorithm": "RS256",
    "clients": [
      {
        "client_id": "abc123def456",
        "client_name": "Test Client 1",
        "public_key_path": "storage/keys/client_001.pub",
        "permissions": [
          "read:health",
          "read:ping",
          "admin:logs"
        ],
        "active": true
      }
    ]
  }
}
```

## Redis Architecture

### Centralized Redis Client System

The service uses a **mode-agnostic** Redis architecture that supports both single-node and cluster deployments transparently:

**Key Features:**
- **Unified Interface**: Same API for single-node and cluster modes
- **Factory Pattern**: Specialized clients for different use cases
- **Connection Pooling**: Optimized connection management with configurable pool sizes
- **Automatic Failover**: Built-in health checks and connection validation

**Redis Configuration by Mode:**

### Single-Node Redis (Default)
```json
{
  "redis": {
    "host": "redis",
    "port": 6379,
    "password": "",
    "db": 0
  }
}
```
*Note: `mode` field is optional - defaults to `"single"` for backward compatibility*

### Cluster Redis (High Availability)
```json
{
  "redis": {
    "mode": "cluster",
    "cluster": {
      "nodes": ["redis-1:6379", "redis-2:6379", "redis-3:6379"],
      "password": ""
    }
  }
}
```

### Sentinel Redis (Auto-failover) - **Not Yet Implemented**
```json
{
  "redis": {
    "mode": "sentinel",
    "sentinel": {
      "master_name": "mymaster",
      "nodes": ["sentinel-1:26379", "sentinel-2:26379"],
      "password": ""
    }
  }
}
```
*Note: Configuration structure is ready, but implementation is pending*

**Mode Comparison:**

| Mode | Use Case | Data Separation | Config Required | Status |
|------|----------|-----------------|-----------------|--------|
| `single` | Development, Production | Redis DB (0-4) | Host + Port only | ✅ **Ready** |
| `cluster` | High Availability | Key prefixes | Node list | ✅ **Ready** |
| `sentinel` | Auto-failover | Redis DB (0-4) | Master + Sentinels | ❌ **Not Implemented** |

**Specialized Clients:**
```go
redis.NewClientForMain()        // General app data (default pool: 300)
redis.NewClientForAsynq()       // Job queue (optimized pool: 200)
redis.NewClientForSessions()    // User sessions (lightweight pool: 50)
redis.NewClientForCache()       // App cache (medium pool: 100)
redis.NewClientForNonceStore()  // Replay protection (small pool: 20)
```

**Database Separation:**
- **Single-node**: Uses Redis databases 0-4 for logical separation
- **Cluster**: Uses key prefixes since cluster mode doesn't support DB selection

## Per-Client Redis Pool Configuration

**Advanced Pool Management**: The service now features per-client Redis pool configuration for optimal performance tuning and resource management.

### Key Features
- ✅ **Client-Specific Pools**: Different pool sizes optimized for each use case
- ✅ **Smart Fallback System**: 4-tier fallback hierarchy for maximum compatibility  
- ✅ **Cluster Compatibility**: Works seamlessly with both single-node and cluster modes
- ✅ **Performance Tuning**: Easy optimization via configuration without code changes
- ✅ **Backward Compatibility**: Legacy `asynq.pool_size` still supported

### Pool Configuration Structure
```json
"redis": {
  "pools": {
    "default": {       // Main operations (300 connections)
      "size": 300,
      "timeout": "5s",
      "dial_timeout": "3s",
      "read_timeout": "2s",
      "write_timeout": "2s",
      "max_lifetime": "30m",
      "idle_timeout": "10m"
    },
    "asynq": {         // Background job processing (200 connections)
      "size": 200,
      "timeout": "10s",
      "max_lifetime": "30m",
      "idle_timeout": "5m"
    },
    "sessions": {      // User sessions (50 connections)
      "size": 50,
      "timeout": "3s",
      "max_lifetime": "15m",
      "idle_timeout": "2m"
    },
    "cache": {         // Application cache (100 connections)
      "size": 100,
      "timeout": "5s",
      "max_lifetime": "20m",
      "idle_timeout": "5m"
    },
    "nonce": {         // Auth nonce storage (20 connections)
      "size": 20,
      "timeout": "2s",
      "max_lifetime": "10m",
      "idle_timeout": "1m"
    }
  }
}
```

### Fallback Hierarchy
1. **Client-Specific Pool**: `redis.pools.{clientType}` (e.g., `redis.pools.asynq`)
2. **Default Pool**: `redis.pools.default` (fallback for all clients)
3. **Legacy Config**: `asynq.pool_size` (backward compatibility)
4. **Safe Defaults**: Hardcoded fallback (size: 10, timeout: 30s)

### Performance Tuning Examples
```json
// High-performance Asynq for heavy job processing
"asynq": {"size": 300, "timeout": "15s"}

// Memory-optimized sessions  
"sessions": {"size": 25, "idle_timeout": "1m"}

// High-performance cache with longer connections
"cache": {"size": 200, "max_lifetime": "1h"}
```

### Benefits
- **🚀 Performance**: Pool size **15 → 200** for Asynq (13x improvement)
- **⚡ Resource Optimization**: Right-sized pools prevent over/under-provisioning
- **🎯 Easy Tuning**: Change config without code deployment
- **🔧 Client-Specific**: Each client optimized for its usage pattern
- **💾 Connection Lifecycle**: Proper MaxLifetime and IdleTimeout management

## Multi Authentication System

The service supports **three authentication methods** with optional replay attack protection: JWT, Signature-based (RSA & HMAC), and Multi-Auth.

### Authentication Method Comparison

**JWT Authentication**
- ✅ **Industry standard** - Widely supported across platforms and libraries
- ✅ **Stateless** - No server-side session storage required
- ✅ **Token-based** - Single token contains all authentication info
- ✅ **Cross-domain friendly** - Easy to use across different services
- ✅ **Built-in expiration** - Automatic token expiry handling
- ⚠️ **Slower verification** - RSA signature verification overhead
- ⚠️ **HTTP-centric** - Primarily designed for HTTP Authorization header

**Signature Authentication**  
- ✅ **100x faster** - HMAC signature verification vs RSA JWT
- ✅ **Transport agnostic** - Works with HTTP, gRPC, WebSocket, any protocol
- ✅ **Request integrity** - Signature covers entire request payload
- ✅ **Enhanced replay protection** - 30-second window + optional nonce
- ✅ **Flexible algorithms** - RSA256, RSA512, HMAC256, HMAC512 options
- ✅ **Real-time security** - Fresh signature for each request
- ⚠️ **More complex** - Requires custom signature generation logic
- ⚠️ **Less standardized** - Custom implementation vs standard JWT

**Multi-Auth**
- ✅ **Best of both worlds** - Use JWT for simplicity, Signature for performance
- ✅ **Client choice** - Clients can choose the method that fits their needs
- ✅ **Gradual migration** - Easy to migrate between authentication methods
- ✅ **Backward compatibility** - Existing clients continue working unchanged

## JWT Authentication (RSA-based)

### Overview
The service uses JWT-based authentication with RSA256 algorithm and public/private key verification. Each client has their own key pair and permissions.

### Permission System
- **Actions**: `create`, `read`, `update`, `delete`, `admin`, `bulk`, `export`
- **Format**: `action:resource` (e.g., `read:health`, `admin:logs`)
- **Wildcards**: 
  - `*:*` = Super admin (all permissions)
  - `*:resource` = All actions for specific resource
  - `action:*` = Specific action for all resources
- **Admin**: `admin:resource` covers all CRUD actions for that resource

### Client Setup

#### 1. Generate RSA Key Pair
```bash
# Generate private key
openssl genrsa -out client_private.pem 2048

# Extract public key
openssl rsa -in client_private.pem -pubout -out client_public.pub
```

#### 2. Add RSA Client to Config
```json
{
  "auth": {
    "clients": [
      {
        "client_id": "your-random-client-id",
        "client_name": "Your Client Name",
        "auth_type": "rsa",
        "key_path": "storage/keys/your_client.pub",
        "permissions": [
          "read:health",
          "read:ping"
        ],
        "active": true
      }
    ]
  }
}
```

#### 3. Place Public Key
```bash
# Copy public key to storage directory
cp client_public.pub storage/keys/your_client.pub
```

### JWT Generation Examples

#### Node.js
```javascript
const jwt = require('jsonwebtoken');
const fs = require('fs');

// Read private key
const privateKey = fs.readFileSync('client_private.pem');

// Create JWT payload (only client_id required)
const payload = {
  client_id: 'your-random-client-id',
  iat: Math.floor(Date.now() / 1000),
  exp: Math.floor(Date.now() / 1000) + (60 * 60) // 1 hour
};

// Sign JWT
const token = jwt.sign(payload, privateKey, { 
  algorithm: 'RS256',
  header: { alg: 'RS256', typ: 'JWT' }
});

console.log('JWT Token:', token);
```

#### Python
```python
import jwt
import json
from datetime import datetime, timedelta

# Read private key
with open('client_private.pem', 'rb') as key_file:
    private_key = key_file.read()

# Create payload (only client_id required)
payload = {
    'client_id': 'your-random-client-id',
    'iat': datetime.utcnow(),
    'exp': datetime.utcnow() + timedelta(hours=1)
}

# Generate JWT
token = jwt.encode(payload, private_key, algorithm='RS256')
print(f'JWT Token: {token}')
```

#### Go
```go
package main

import (
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "fmt"
    "io/ioutil"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    ClientID string `json:"client_id"`
    jwt.RegisteredClaims
}

func main() {
    // Read private key
    keyData, _ := ioutil.ReadFile("client_private.pem")
    block, _ := pem.Decode(keyData)
    privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
    
    // Create claims (only client_id required)
    claims := Claims{
        ClientID: "your-random-client-id",
        RegisteredClaims: jwt.RegisteredClaims{
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
        },
    }
    
    // Create and sign token
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    tokenString, _ := token.SignedString(privateKey)
    
    fmt.Printf("JWT Token: %s\n", tokenString)
}
```

### Making Authenticated Requests

```bash
# Set your JWT token
export JWT_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

# Make authenticated request
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:8080/v1/health

# Example response
{
  "success": true,
  "code": 0,
  "data": {
    "status": "healthy",
    "services": {...}
  },
  "message": "Successful",
  "request_id": "req-1234567890"
}
```

## Signature Authentication with Replay Protection

### Overview
Transport-agnostic authentication using request signatures. Supports both RSA (public/private key) and HMAC (shared secret) algorithms with optional nonce-based replay protection.


### Security Features
- **30-second timestamp window** - Minimizes replay attack window
- **Optional nonce support** - 100% replay attack prevention when used
- **Memory-efficient cleanup** - Automatic nonce cleanup every 5 minutes
- **Context-managed worker** - Graceful shutdown of cleanup processes

### Client Setup

#### Option 1: RSA Signature Client
```json
{
  "auth": {
    "algorithm": "RS256",
    "clients": [
      {
        "client_id": "your-random-client-id",
        "client_name": "Your RSA Client",
        "auth_type": "rsa",
        "key_path": "storage/keys/your_client.pub",
        "permissions": ["read:health"],
        "active": true
      }
    ]
  }
}
```

#### Option 2: HMAC Signature Client  
```json
{
  "auth": {
    "algorithm": "HS256",
    "clients": [
      {
        "client_id": "your-random-client-id", 
        "client_name": "Your HMAC Client",
        "auth_type": "hmac",
        "secret_key": "780bc02a1031f0a3577c93ef457f345b3624370754d3f5a377b39899cffd50ef",
        "permissions": ["*:*"],
        "active": true
      }
    ]
  }
}
```

### CLI Client Management
Comprehensive CLI-based client management system for security:

```bash
# Create new HMAC client
./insight-collector client create --name "Test Client" --type hmac --permissions "read:health,admin:logs"

# Create new RSA client  
./insight-collector client create --name "RSA Client" --type rsa --key-path "client.pub" --permissions "read:health"

# List all clients
./insight-collector client list

# Show client details
./insight-collector client show abc123def456

# Revoke/activate client
./insight-collector client revoke abc123def456
./insight-collector client activate abc123def456

# Regenerate HMAC secret key
./insight-collector client regenerate abc123def456

# Delete client permanently
./insight-collector client delete abc123def456 --force

# Generate test signatures
./insight-collector client generatesign abc123def456                    # Without nonce
./insight-collector client generatesign abc123def456 --with-nonce       # With nonce
./insight-collector client generatesign abc123def456 --method POST --path /v1/ping
```

### Signature Generation Examples

#### Node.js (RSA)
```javascript
const crypto = require('crypto');
const fs = require('fs');

// Payload structure (canonical JSON) - nonce is optional
const payload = {
  client_id: 'your-random-client-id',
  timestamp: Math.floor(Date.now() / 1000),
  nonce: crypto.randomBytes(16).toString('hex'), // Optional for replay protection
  method: 'GET',
  path: '/v1/health',
  body: ''
};

// Generate signature
const privateKey = fs.readFileSync('client_private.pem');
const payloadStr = JSON.stringify(payload); // Canonical JSON
const signature = crypto.sign('sha256', Buffer.from(payloadStr), {
  key: privateKey,
  padding: crypto.constants.RSA_PKCS1_PADDING
});
const base64Signature = signature.toString('base64');

console.log('Headers:');
console.log('X-Client-ID:', payload.client_id);
console.log('X-Timestamp:', payload.timestamp);
console.log('X-Nonce:', payload.nonce); // Optional
console.log('X-Signature:', base64Signature);
```

#### Node.js (HMAC)
```javascript
const crypto = require('crypto');

// Payload structure (canonical JSON) - nonce is optional
const payload = {
  client_id: 'your-random-client-id',
  timestamp: Math.floor(Date.now() / 1000),
  nonce: crypto.randomBytes(16).toString('hex'), // Optional for replay protection
  method: 'GET', 
  path: '/v1/health',
  body: ''
};

// Generate HMAC signature
const secretKey = 'your-super-secret-key-string';
const payloadStr = JSON.stringify(payload);
const hmac = crypto.createHmac('sha256', secretKey);
hmac.update(payloadStr);
const signature = hmac.digest('base64');

console.log('Headers:');
console.log('X-Client-ID:', payload.client_id);
console.log('X-Timestamp:', payload.timestamp);
console.log('X-Nonce:', payload.nonce); // Optional
console.log('X-Signature:', signature);
```

#### Python (RSA)
```python
import json
import time
import base64
import secrets
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding

# Payload structure - nonce is optional
payload = {
    "client_id": "your-random-client-id",
    "timestamp": int(time.time()),
    "nonce": secrets.token_hex(16),  # Optional for replay protection
    "method": "GET",
    "path": "/v1/health", 
    "body": ""
}

# Load private key
with open("client_private.pem", "rb") as key_file:
    private_key = serialization.load_pem_private_key(
        key_file.read(), password=None
    )

# Generate signature
payload_str = json.dumps(payload, separators=(',', ':'), sort_keys=True)
signature = private_key.sign(
    payload_str.encode('utf-8'),
    padding.PKCS1v15(),
    hashes.SHA256()
)
base64_signature = base64.b64encode(signature).decode('utf-8')

print(f"X-Client-ID: {payload['client_id']}")
print(f"X-Timestamp: {payload['timestamp']}")
print(f"X-Nonce: {payload['nonce']}")  # Optional
print(f"X-Signature: {base64_signature}")
```

#### Python (HMAC)
```python
import json
import time
import hmac
import hashlib
import base64
import secrets

# Payload structure - nonce is optional
payload = {
    "client_id": "your-random-client-id",
    "timestamp": int(time.time()),
    "nonce": secrets.token_hex(16),  # Optional for replay protection
    "method": "GET",
    "path": "/v1/health",
    "body": ""
}

# Generate HMAC signature
secret_key = "your-super-secret-key-string"
payload_str = json.dumps(payload, separators=(',', ':'), sort_keys=True)
signature = hmac.new(
    secret_key.encode('utf-8'),
    payload_str.encode('utf-8'),
    hashlib.sha256
).digest()
base64_signature = base64.b64encode(signature).decode('utf-8')

print(f"X-Client-ID: {payload['client_id']}")
print(f"X-Timestamp: {payload['timestamp']}")
print(f"X-Nonce: {payload['nonce']}")  # Optional
print(f"X-Signature: {base64_signature}")
```

### Making Signature Requests

```bash
# Set signature components
export CLIENT_ID="your-random-client-id"
export TIMESTAMP="1640995200"
export NONCE="1a2b3c4d5e6f7890abcdef1234567890"  # Optional
export SIGNATURE="generated_base64_signature"

# Make signature request without nonce (backward compatible)
curl -H "X-Client-ID: $CLIENT_ID" \
     -H "X-Timestamp: $TIMESTAMP" \
     -H "X-Signature: $SIGNATURE" \
     http://localhost:8080/v1/health/signature

# Make signature request with nonce (enhanced security)
curl -H "X-Client-ID: $CLIENT_ID" \
     -H "X-Timestamp: $TIMESTAMP" \
     -H "X-Nonce: $NONCE" \
     -H "X-Signature: $SIGNATURE" \
     http://localhost:8080/v1/health/signature

# Example response
{
  "success": true,
  "code": 0,
  "data": {
    "status": "healthy",
    "services": {...}
  },
  "message": "Successful", 
  "request_id": "req-1234567890"
}
```

## Multi-Auth (JWT + Signature)

### Overview
Single endpoints that accept **both JWT and Signature** authentication automatically.

### Usage Examples

```bash
# Same endpoint, different auth methods

# Using JWT
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/v1/health

# Using Signature
curl -H "X-Client-ID: your-client-id" \
     -H "X-Timestamp: 1640995200" \
     -H "X-Signature: base64_signature" \
     http://localhost:8080/v1/health
```

## API Endpoints by Auth Type

### Public Endpoints (No Auth)
```bash
curl http://localhost:8080/v1/health/live   # Liveness probe
curl http://localhost:8080/v1/health/ready  # Readiness probe
```

### JWT-Only Endpoints
```bash 
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/v1/health/jwt
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/v1/ping/jwt
```

### Signature-Only Endpoints
```bash
curl -H "X-Client-ID: client" -H "X-Timestamp: time" -H "X-Signature: sig" \
  http://localhost:8080/v1/health/signature
curl -H "X-Client-ID: client" -H "X-Timestamp: time" -H "X-Signature: sig" \
  http://localhost:8080/v1/ping/signature
```

### Multi-Auth Endpoints (JWT or Signature)
```bash
# Works with either JWT or Signature
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/v1/health
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/v1/ping

# OR

curl -H "X-Client-ID: client" -H "X-Timestamp: time" -H "X-Signature: sig" \
  http://localhost:8080/v1/health
curl -H "X-Client-ID: client" -H "X-Timestamp: time" -H "X-Signature: sig" \
  http://localhost:8080/v1/ping
```

## Standardized Error Codes

### Error Code Format
The API uses **5-digit standardized error codes** with automatic HTTP status mapping:

**Format:** `XYZABC`
- **X** = Category (4=client error, 5=server error)
- **YZ** = HTTP status code without first digit (00=400, 01=401, 04=404)
- **ABC** = Specific error within category (000=generic, 001=first specific)

### Common Error Codes
```bash
# Client Errors (4xxxx)
40000 - Bad Request (generic)
40001 - Invalid JSON payload
40002 - Validation failed

# Authentication Errors (41xxx) 
41000 - Unauthorized (generic)
41001 - Missing authentication
41002 - Invalid JWT token
41004 - Invalid signature
41008 - Nonce replay attack detected

# Permission Errors (43xxx)
43000 - Forbidden (generic) 
43001 - Insufficient permissions

# Not Found Errors (44xxx)
44000 - Not found (generic)
44001 - Resource not found

# Server Errors (5xxxx)
50000 - Internal server error
50001 - Database error
50002 - InfluxDB error
50003 - Redis error
```

### Error Response Examples

#### Missing Authentication
```json
{
  "success": false,
  "code": 41001,
  "data": null,
  "message": "Authentication required: provide either Bearer token or X-Signature",
  "request_id": "req-1234567890"
}
```

#### Invalid Signature
```json
{
  "success": false,
  "code": 41004,
  "data": null,
  "message": "Invalid signature",
  "request_id": "req-1234567890"
}
```

#### Insufficient Permissions
```json
{
  "success": false,
  "code": 43001,
  "data": null,
  "message": "Insufficient permissions",
  "request_id": "req-1234567890"
}
```

### Using Error Codes in Development
```go
// Recommended - using standardized error codes
return response.FailWithCode(c, constants.CodeInvalidJSON)
return response.FailWithCode(c, constants.CodeInsufficientPerms)

// Alternative - custom message with standard code
return response.FailWithCodeAndMessage(c, constants.CodeValidationFailed, "Email format is invalid")

// Legacy - still supported but not recommended
return response.Fail(c, 400, 40001, "Invalid JSON payload")
```

## Key Features

### Authentication & Security
- **Multi Authentication**: JWT, Signature (RSA & HMAC), and Multi-Auth support
- **Enhanced Replay Protection**: 30-second window + optional nonce for 100% prevention
- **CLI Client Management**: Zero-downtime client management with dual update system
- **Transport Agnostic**: Signature auth works with HTTP, gRPC, WebSocket, etc.
- **Performance Options**: JWT (secure) vs HMAC (100x faster) vs RSA (balanced)
- **CRUD Permissions**: Fine-grained access control with wildcards
- **Algorithm Flexibility**: RS256, RS512, HS256, HS512 support

### InfluxDB Cursor-Based Pagination
- **True Server-Side Pagination**: RFC3339 timestamp cursors for efficient navigation of 100k+ records
- **Dynamic Filtering System**: Configurable tag/field validation with security controls
- **Date Range Support**: Exact dates and ranges (YYYY-MM-DD format)
- **Safety Limits**: 10x multiplier (50-1000 cap) prevents catastrophic data transfer on first page
- **Memory-Efficient Processing**: Automatic cleanup of internal InfluxDB fields
- **Entity-Based Configuration**: Reusable query builder architecture with dependency injection
- **Type-Safe Interfaces**: Import cycle-free design with proper error handling
- **Production Optimizations**: 60-second timeouts, connection pooling, cursor efficiency

### MaxMind GeoIP System
- **Thread-safe GeoIP Lookups**: City and ASN data with zero-downtime database reloads
- **Automatic Database Downloader**: SHA256-verified downloads with retry mechanisms
- **Conditional Downloads**: Only download when databases are newer (Last-Modified headers)
- **Periodic Update Checking**: Configurable interval checking (default 1 hour)
- **Basic Authentication**: MaxMind account credentials with environment variable support
- **Graceful Fallback**: Service continues with default values when databases unavailable
- **CLI Management**: Complete database management via command line interface
- **Database Versioning**: Metadata tracking for database versions and update history

### Job Queue System
- **Auto-generation from Registry**: Workers automatically created from registered jobs
- **Type-safe Queue Constants**: Queue names as constants with validation (`critical`, `default`, `low`)
- **Single Registration Point**: Jobs registered once with explicit queue assignment
- **Redis Persistence**: Worker configuration persisted and auto-loaded
- **Advanced Worker Management**: JSON-formatted CLI with incremental task addition and auto-generation
- **Centralized Job Dispatching**: Structured `Payload` type with automatic duplicate handling
- **Auto-routing**: Jobs automatically routed based on registry configuration
- **Smart Percentage Allocation**: Queues get intended percentages when jobs are assigned

### Development & Operations
- **Versioned API**: v1, v2 route separation with registry pattern
- **Centralized Response**: Request-ID auto-included in all responses
- **Input Validation**: Echo validator with struct tags
- **Scoped Logging**: Environment-based logger optimization with scope support
- **Graceful Shutdown**: Safe stop with Ctrl+C and resource cleanup
- **Hot Reload**: Development with Air
- **Zero Downtime**: Overseer for HTTP server restarts
- **Structured Logging**: Zerolog with timezone support and field ordering
- **Standardized Error Codes**: 5-digit categorized error codes with HTTP status mapping

## Production Deployment

### Systemd Integration

**HTTP Server Service:**
```bash
# /etc/systemd/system/insight-server.service
[Unit]
Description=Insight Collector HTTP Server
After=network.target

[Service]
Type=simple
User=insight
WorkingDirectory=/opt/insight-collector
ExecStart=/opt/insight-collector/insight-collector serve
ExecReload=/bin/kill -USR2 $MAINPID
Restart=always
RestartSec=5

# Graceful shutdown settings
TimeoutStopSec=45
KillMode=mixed
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

**Background Worker Service:**
```bash
# /etc/systemd/system/insight-worker.service
[Unit]
Description=Insight Collector Background Worker
After=network.target redis.service

[Service]
Type=simple
User=insight
WorkingDirectory=/opt/insight-collector
ExecStart=/opt/insight-collector/insight-collector worker start
ExecReload=/bin/kill -USR2 $MAINPID
Restart=always
RestartSec=5

# Graceful shutdown settings
TimeoutStopSec=45
KillMode=mixed
KillSignal=SIGTERM

[Install]
WantedBy=multi-user.target
```

**Enable Services:**
```bash
sudo systemctl enable insight-server.service
sudo systemctl enable insight-worker.service
sudo systemctl start insight-server.service
sudo systemctl start insight-worker.service
```

### Graceful Restart

```bash
# Zero-downtime restart via overseer
kill -USR2 $(pgrep -f "insight-collector serve")   # HTTP server (:3000)
kill -USR2 $(pgrep -f "insight-collector worker")  # Worker (:3001)
```

### Service Management

```bash
# Check status
sudo systemctl status insight-server
sudo systemctl status insight-worker

# View logs
sudo journalctl -u insight-server -f
sudo journalctl -u insight-worker -f

# Restart services
sudo systemctl restart insight-server
sudo systemctl restart insight-worker
```

## Development Flow

1. **Add API feature** → entities → services → handlers → routes
2. **Add background job** → job handler → register in `jobs/registry.go` with queue constant → workers auto-generate
3. **Test locally** → `air` for hot reload, workers auto-start with generated config
4. **Deploy** → Docker Compose + Systemd for production reliability

### Zero-Configuration Job Processing
- **First time**: Jobs auto-generate workers, immediately processable
- **Add new job**: Workers dynamically adjust percentages
- **Manual override**: CLI commands for custom allocation (persisted to Redis)
