#!/bin/bash

# Realtime Monitoring Script for Stress Test
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')
LOG_FILE="results/monitoring_${TIMESTAMP}.log"

# Create results directory if it doesn't exist
mkdir -p results

# Function to output to both console and log file
output() {
    echo "$1" | tee -a "$LOG_FILE"
}

output "🔍 Starting Realtime Monitoring for Stress Test..."
output "📊 Monitoring Duration: 16 minutes (960 seconds)"
output "⏰ Started at: $(date)"
output "📝 Log file: $LOG_FILE"
output "==========================================="

# Get application PID
APP_PID=$(docker exec insight-collector pgrep -f "main dev")
if [ -z "$APP_PID" ]; then
    output "❌ Application process not found!"
    exit 1
fi

output "📱 Monitoring Process: $APP_PID"
output "==========================================="

# Function to log with timestamp
log_with_time() {
    output "[$(date '+%H:%M:%S')] $1"
}

# Function to get memory info
get_memory_info() {
    docker exec insight-collector cat /proc/$APP_PID/status 2>/dev/null | grep -E "(VmRSS|VmSize)" | tr -d '\t' | tr '\n' ' '
}

# Function to get connection count
get_connection_count() {
    docker exec insight-collector netstat -an 2>/dev/null | grep ":8080" | grep ESTABLISHED | wc -l
}

# Function to get Redis info
get_redis_info() {
    docker exec insight-collector-redis redis-cli info clients 2>/dev/null | grep connected_clients | cut -d: -f2 | tr -d '\r'
}

# Function to check application health
check_health() {
    curl -s -w "%{time_total}" -o /dev/null http://localhost:8080/v1/health/live 2>/dev/null || echo "TIMEOUT"
}

# Function to get container stats
get_container_stats() {
    docker stats insight-collector --no-stream --format "{{.CPUPerc}},{{.MemUsage}}" 2>/dev/null
}

# Function to get Redis memory info
get_redis_memory() {
    docker exec insight-collector-redis redis-cli info memory 2>/dev/null | grep -E "(used_memory_human|maxmemory_human)" | tr '\n' ' ' | sed 's/used_memory_human:/Mem:/' | sed 's/maxmemory_human:/Max:/'
}

# Function to get Redis pool stats
get_redis_pool_stats() {
    docker exec insight-collector-redis redis-cli info stats 2>/dev/null | grep -E "(total_connections_received|instantaneous_ops_per_sec)" | tr '\n' ' ' | sed 's/total_connections_received:/Total:/' | sed 's/instantaneous_ops_per_sec:/OPS:/'
}

# Function to get Asynq queue details
get_asynq_queues() {
    # Discovery mode: Find actual queue keys first
    local queue_keys=$(docker exec insight-collector-redis redis-cli keys "asynq:*" 2>/dev/null | grep -E "(critical|low|default)" | head -3)
    
    # Try different common patterns
    local critical=$(docker exec insight-collector-redis redis-cli llen "asynq:queues:critical" 2>/dev/null || \
                     docker exec insight-collector-redis redis-cli llen "asynq:critical" 2>/dev/null || echo "0")
    local low=$(docker exec insight-collector-redis redis-cli llen "asynq:queues:low" 2>/dev/null || \
                docker exec insight-collector-redis redis-cli llen "asynq:low" 2>/dev/null || echo "0")
    local default=$(docker exec insight-collector-redis redis-cli llen "asynq:queues:default" 2>/dev/null || \
                    docker exec insight-collector-redis redis-cli llen "asynq:default" 2>/dev/null || echo "0")
    
    echo "Critical:$critical Low:$low Default:$default"
}

# Function to get Asynq worker stats  
get_asynq_workers() {
    # Safe mode: Only get metrics that definitely work
    local asynq_keys=$(docker exec insight-collector-redis redis-cli keys "asynq:*" 2>/dev/null | wc -l)
    # Check for Air dev mode process (contains both server and worker)
    local main_process=$(docker exec insight-collector pgrep -f "main dev" 2>/dev/null | wc -l)
    
    # Get configured worker concurrency from config (simplified approach)
    local worker_config=$(docker exec insight-collector cat .config.json 2>/dev/null | jq -r '.asynq.concurrency' 2>/dev/null || echo "N/A")
    
    # Simple check: if we have asynq keys and main process, workers are running
    if [ "$asynq_keys" -gt 0 ] && [ "$main_process" -gt 0 ]; then
        echo "Status:RUNNING Process:$main_process Keys:$asynq_keys Workers:$worker_config"
    else
        echo "Status:STOPPED Process:$main_process Keys:$asynq_keys Workers:$worker_config"
    fi
}

# Function to get Go runtime stats
get_go_runtime() {
    local goroutines=$(docker exec insight-collector cat /proc/$APP_PID/status 2>/dev/null | grep "Threads:" | awk '{print $2}')
    local fds=$(docker exec insight-collector ls /proc/$APP_PID/fd 2>/dev/null | wc -l)
    echo "Goroutines:$goroutines FDs:$fds"
}

# Function to get HTTP server detailed stats
get_http_detailed() {
    local listen=$(docker exec insight-collector netstat -an 2>/dev/null | grep ":8080" | grep LISTEN | wc -l)
    local timewait=$(docker exec insight-collector netstat -an 2>/dev/null | grep ":8080" | grep TIME_WAIT | wc -l)
    local closewait=$(docker exec insight-collector netstat -an 2>/dev/null | grep ":8080" | grep CLOSE_WAIT | wc -l)
    echo "Listen:$listen TimeWait:$timewait CloseWait:$closewait"
}

# Function to get InfluxDB write performance
get_influxdb_metrics() {
    # Check response time to InfluxDB health
    local influx_health=$(curl -s -w "%{time_total}" -o /dev/null http://localhost:8086/health 2>/dev/null || echo "TIMEOUT")
    
    # Get user_activities record count (recent data)
    local record_count=$(docker exec influxdb2-oss influx query 'from(bucket: "insight_collector") |> range(start: -1h) |> filter(fn: (r) => r["_measurement"] == "user_activities") |> filter(fn: (r) => r["_field"] == "request_id") |> count()' --org insight --token my-super-secret-auth-token-influxdb2-oss 2>/dev/null | tail -n1 | awk '{print $NF}' || echo "0")
    
    echo "Health:${influx_health}s Records:$record_count"
}

# Function to get server debug info with HTTP optimization metrics
get_server_debug() {
    local debug_response=$(curl -s http://localhost:8080/debug/connections 2>/dev/null)
    if [ $? -eq 0 ]; then
        # Extract connection stats using python for reliable JSON parsing
        local connection_stats=$(echo "$debug_response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    stats = data['data']['connection_stats']
    runtime = data['data']['runtime_info']
    print(f\"Active:{stats['active_connections']}/{stats['max_connections']} Util:{stats['utilization_pct']:.1f}% Total:{stats['total_connections']} GOMAXPROCS:{runtime['gomaxprocs']}\")
except:
    print('ParseError')
" 2>/dev/null)
        echo "${connection_stats:-ParseError}"
    else
        echo "ConnectionTimeout"
    fi
}

# Function to get system file descriptor limits
get_system_limits() {
    local ulimit_n=$(docker exec insight-collector sh -c 'ulimit -n' 2>/dev/null || echo "N/A")
    local proc_limits=$(docker exec insight-collector cat /proc/$APP_PID/limits 2>/dev/null | grep "Max open files" | awk '{print $4}' || echo "N/A")
    echo "Ulimit:$ulimit_n ProcLimit:$proc_limits"
}

# Function to get queue status (legacy - kept for compatibility)
get_queue_status() {
    docker exec insight-collector-redis redis-cli llen "asynq:{insight_collector}:pending" 2>/dev/null || echo "N/A"
}

# Monitoring loop
DURATION=720  # 12 minutes
INTERVAL=10   # Check every n seconds
ITERATIONS=$((DURATION / INTERVAL))

output "🚀 Starting monitoring loop (checking every ${INTERVAL} seconds for ${ITERATIONS} iterations)"
output "==========================================="

for i in $(seq 1 $ITERATIONS); do
    ELAPSED=$((i * INTERVAL))
    REMAINING=$((DURATION - ELAPSED))
    
    log_with_time "=== Check $i/$ITERATIONS (${ELAPSED}s elapsed, ${REMAINING}s remaining) ==="
    
    # Application Process Check
    if ! docker exec insight-collector kill -0 $APP_PID 2>/dev/null; then
        log_with_time "❌ APPLICATION CRASHED! Process $APP_PID not found!"
        break
    fi
    
    # === SYSTEM METRICS ===
    MEMORY=$(get_memory_info)
    log_with_time "📊 Memory: $MEMORY"
    
    CONTAINER_STATS=$(get_container_stats)
    if [ ! -z "$CONTAINER_STATS" ]; then
        CPU=$(echo $CONTAINER_STATS | cut -d, -f1)
        MEM=$(echo $CONTAINER_STATS | cut -d, -f2)
        log_with_time "🖥️  Container: CPU=$CPU MEM=$MEM"
    fi
    
    # === GO RUNTIME METRICS ===
    GO_RUNTIME=$(get_go_runtime)
    log_with_time "⚙️  Go Runtime: $GO_RUNTIME"
    
    # === HTTP SERVER METRICS ===
    CONNECTIONS=$(get_connection_count)
    log_with_time "🌐 HTTP Connections: $CONNECTIONS"
    
    HTTP_DETAILED=$(get_http_detailed)
    log_with_time "🔗 HTTP States: $HTTP_DETAILED"
    
    # === REDIS METRICS ===
    REDIS_CONN=$(get_redis_info)
    log_with_time "🔄 Redis Connections: $REDIS_CONN"
    
    REDIS_MEMORY=$(get_redis_memory)
    log_with_time "💾 Redis Memory: $REDIS_MEMORY"
    
    REDIS_POOL=$(get_redis_pool_stats)
    log_with_time "🏊 Redis Pool: $REDIS_POOL"
    
    # === ASYNQ METRICS ===
    ASYNQ_QUEUES=$(get_asynq_queues)
    log_with_time "📋 Asynq Queues: $ASYNQ_QUEUES"
    
    ASYNQ_WORKERS=$(get_asynq_workers)
    log_with_time "👷 Asynq Workers: $ASYNQ_WORKERS"
    
    # === INFLUXDB METRICS ===
    INFLUX_METRICS=$(get_influxdb_metrics)
    log_with_time "📈 InfluxDB: $INFLUX_METRICS"
    
    # === HTTP OPTIMIZATION METRICS ===
    SERVER_DEBUG=$(get_server_debug)
    log_with_time "🚀 HTTP Optimization: $SERVER_DEBUG"
    
    # === SYSTEM LIMITS ===
    SYSTEM_LIMITS=$(get_system_limits)
    log_with_time "⚙️  System Limits: $SYSTEM_LIMITS"
    
    # === HEALTH CHECK ===
    HEALTH_TIME=$(check_health)
    if [ "$HEALTH_TIME" = "TIMEOUT" ]; then
        log_with_time "⚠️  Health Check: TIMEOUT/ERROR"
    else
        log_with_time "❤️  Health Check: ${HEALTH_TIME}s"
    fi
    
    # === PAGINATION PERFORMANCE TEST ===
    # Test pagination endpoint performance with growing dataset
    PAGINATION_TIME=$(curl -s -w "%{time_total}" -o /dev/null -X POST http://localhost:8080/v1/user-activities/list -H "Content-Type: application/json" -d '{"length": 10, "direction": "next"}' 2>/dev/null || echo "TIMEOUT")
    if [ "$PAGINATION_TIME" = "TIMEOUT" ]; then
        log_with_time "⚠️  Pagination Test: TIMEOUT/ERROR"
    else
        log_with_time "📄 Pagination Test: ${PAGINATION_TIME}s"
    fi
    
    # === ALERT CHECKS ===
    # HTTP Connection Alert (updated for 800 max connections)
    if [ "$CONNECTIONS" -gt 640 ]; then
        log_with_time "🚨 CRITICAL: HTTP Connections > 640: $CONNECTIONS (approaching limit 800)"
    elif [ "$CONNECTIONS" -gt 400 ]; then
        log_with_time "⚠️  WARNING: HTTP Connections > 400: $CONNECTIONS"
    fi
    
    # HTTP Optimization Alert (check if connection limiting is working)
    if echo "$SERVER_DEBUG" | grep -q "Util:"; then
        UTILIZATION=$(echo "$SERVER_DEBUG" | grep -o "Util:[0-9.]*%" | cut -d: -f2 | cut -d% -f1)
        if [ "$(echo "$UTILIZATION" | cut -d. -f1)" -gt 80 ]; then
            log_with_time "🚨 CRITICAL: HTTP Utilization > 80%: ${UTILIZATION}%"
        elif [ "$(echo "$UTILIZATION" | cut -d. -f1)" -gt 60 ]; then
            log_with_time "⚠️  WARNING: HTTP Utilization > 60%: ${UTILIZATION}%"
        fi
    fi
    
    # Extract queue totals for alerting
    CRITICAL_Q=$(echo $ASYNQ_QUEUES | cut -d' ' -f1 | cut -d: -f2)
    LOW_Q=$(echo $ASYNQ_QUEUES | cut -d' ' -f2 | cut -d: -f2)
    TOTAL_QUEUE=$((CRITICAL_Q + LOW_Q))
    
    if [ "$TOTAL_QUEUE" -gt 100 ]; then
        log_with_time "🚨 CRITICAL: Queue Size > 100: $TOTAL_QUEUE"
    elif [ "$TOTAL_QUEUE" -gt 50 ]; then
        log_with_time "⚠️  WARNING: Queue Size > 50: $TOTAL_QUEUE"
    fi
    
    # Redis connection alerting
    if [ "$REDIS_CONN" -gt 90 ]; then
        log_with_time "🚨 CRITICAL: Redis Connections > 90: $REDIS_CONN"
    elif [ "$REDIS_CONN" -gt 70 ]; then
        log_with_time "⚠️  WARNING: Redis Connections > 70: $REDIS_CONN"
    fi
    
    # Health check alerting (using basic comparison for portability)
    if [ "$HEALTH_TIME" != "TIMEOUT" ]; then
        # Convert to milliseconds using shell arithmetic (basic)
        HEALTH_FLOAT=$(echo "$HEALTH_TIME" | cut -d. -f1)
        if [ "$HEALTH_FLOAT" -gt 5 ]; then
            log_with_time "🚨 CRITICAL: Health Check > 5s: ${HEALTH_TIME}s"
        elif [ "$HEALTH_FLOAT" -gt 1 ]; then
            log_with_time "⚠️  WARNING: Health Check > 1s: ${HEALTH_TIME}s"
        fi
    fi
    
    # Pagination performance alerting
    if [ "$PAGINATION_TIME" != "TIMEOUT" ]; then
        # Convert to seconds using shell arithmetic (basic)
        PAGINATION_FLOAT=$(echo "$PAGINATION_TIME" | cut -d. -f1)
        if [ "$PAGINATION_FLOAT" -gt 2 ]; then
            log_with_time "🚨 CRITICAL: Pagination > 2s: ${PAGINATION_TIME}s (possible series explosion!)"
        elif [ "$PAGINATION_FLOAT" -gt 1 ]; then
            log_with_time "⚠️  WARNING: Pagination > 1s: ${PAGINATION_TIME}s"
        fi
    fi
    
    output ""
    sleep $INTERVAL
done

log_with_time "✅ Monitoring completed!"
log_with_time "📊 FINAL COMPREHENSIVE REPORT:"
log_with_time "================================="

# System State
log_with_time "💻 SYSTEM: $(get_memory_info) | $(get_go_runtime)"
log_with_time "🌐 HTTP: Conn:$(get_connection_count) | $(get_http_detailed)"
log_with_time "🔄 REDIS: Conn:$(get_redis_info) | $(get_redis_memory) | $(get_redis_pool)"
log_with_time "📋 ASYNQ: $(get_asynq_queues) | $(get_asynq_workers)"
log_with_time "📈 INFLUX: $(get_influxdb_metrics)"
log_with_time "🚀 HTTP OPT: $(get_server_debug)"
log_with_time "⚙️  LIMITS: $(get_system_limits)"
log_with_time "❤️  HEALTH: $(check_health)s"
log_with_time "📄 PAGINATION: $(curl -s -w "%{time_total}" -o /dev/null -X POST http://localhost:8080/v1/user-activities/list -H "Content-Type: application/json" -d '{"length": 10, "direction": "next"}' 2>/dev/null || echo "TIMEOUT")s"

output "==========================================="
output "📝 Log file: $LOG_FILE"
output "🏁 Monitoring finished at: $(date)"