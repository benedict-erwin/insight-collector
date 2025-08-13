#!/bin/bash

# Enhanced Forensic Monitoring for Asynq Bottleneck Analysis
# This script provides comprehensive monitoring and profiling for deep dive analysis

set -e

# Configuration
RESULTS_DIR="results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_FILE="$RESULTS_DIR/asynq_forensic_${TIMESTAMP}.log"
PPROF_DIR="$RESULTS_DIR/pprof_${TIMESTAMP}"
TRACES_DIR="$RESULTS_DIR/traces_${TIMESTAMP}"

# Monitoring settings
MONITORING_DURATION=960  # 16 minutes (matches stress test)
CHECK_INTERVAL=10        # Every 10 seconds for high resolution
PPROF_INTERVAL=20        # pprof collection every 20 seconds
TRACE_INTERVAL=60        # detailed traces every minute
TOTAL_CHECKS=$((MONITORING_DURATION / CHECK_INTERVAL))

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

print_info() { echo -e "${BLUE}🔬 $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_forensic() { echo -e "${PURPLE}🧪 $1${NC}"; }
print_trace() { echo -e "${CYAN}📊 $1${NC}"; }

# Create directories
mkdir -p "$PPROF_DIR"/{cpu,heap,goroutine,mutex,allocs,block}
mkdir -p "$TRACES_DIR"/{redis,asynq,http,system}

# Initialize logging
exec > >(tee -a "$LOG_FILE") 2>&1

print_forensic "ENHANCED ASYNQ FORENSIC ANALYSIS STARTING"
echo "=================================================="
echo "📊 Duration: ${MONITORING_DURATION}s (${TOTAL_CHECKS} checks every ${CHECK_INTERVAL}s)"
echo "🎯 Focus: Asynq bottleneck root cause identification"
echo "📁 pprof profiles: $PPROF_DIR/"
echo "📁 Trace data: $TRACES_DIR/"
echo "📝 Forensic log: $LOG_FILE"
echo "⏰ Started at: $(date)"
echo "=================================================="
echo "🚀 Monitoring PID: $$"
echo "=================================================="

# Signal handlers for graceful shutdown
cleanup_on_exit() {
    local exit_code=$?
    print_warning "Received interrupt signal. Cleaning up forensic monitoring..."
    
    # Kill background processes
    if [[ -n "$PPROF_PID" ]]; then
        kill $PPROF_PID 2>/dev/null || true
        print_info "Stopped pprof collection"
    fi
    
    if [[ -n "$REDIS_MONITOR_PID" ]]; then
        kill $REDIS_MONITOR_PID 2>/dev/null || true
        print_info "Stopped Redis monitoring"
    fi
    
    print_success "Forensic data saved to:"
    echo "  • Main log: $LOG_FILE"
    echo "  • pprof profiles: $PPROF_DIR/"
    echo "  • Trace data: $TRACES_DIR/"
    echo "  • Analysis ready for deep dive!"
    
    exit $exit_code
}

trap cleanup_on_exit SIGINT SIGTERM

# Background pprof collection with detailed profiling
start_pprof_collection() {
    print_forensic "Starting comprehensive pprof collection..."
    
    (
        local pprof_count=0
        while true; do
            local ts=$(date +"%H%M%S")
            pprof_count=$((pprof_count + 1))
            
            echo "[PPROF-$pprof_count] Collecting profiles at $ts..."
            
            # CPU Profile (20 second sampling for detailed analysis)
            {
                timeout 25 curl -s "http://localhost:9080/debug/pprof/profile?seconds=20" \
                    -o "$PPROF_DIR/cpu/cpu_${ts}.prof" 2>/dev/null
                echo "[PPROF] CPU profile saved: cpu_${ts}.prof"
            } &
            
            # Memory Heap Profile (detailed heap analysis)
            {
                curl -s "http://localhost:9080/debug/pprof/heap" \
                    -o "$PPROF_DIR/heap/heap_${ts}.prof" 2>/dev/null
                echo "[PPROF] Heap profile saved: heap_${ts}.prof"
            } &
            
            # Goroutine Profile (detect blocking/deadlock)
            {
                curl -s "http://localhost:9080/debug/pprof/goroutine" \
                    -o "$PPROF_DIR/goroutine/goroutine_${ts}.prof" 2>/dev/null
                echo "[PPROF] Goroutine profile saved: goroutine_${ts}.prof"
            } &
            
            # Mutex Profile (contention analysis)
            {
                curl -s "http://localhost:9080/debug/pprof/mutex" \
                    -o "$PPROF_DIR/mutex/mutex_${ts}.prof" 2>/dev/null
                echo "[PPROF] Mutex profile saved: mutex_${ts}.prof"
            } &
            
            # Memory Allocation Profile
            {
                curl -s "http://localhost:9080/debug/pprof/allocs" \
                    -o "$PPROF_DIR/allocs/allocs_${ts}.prof" 2>/dev/null
                echo "[PPROF] Allocs profile saved: allocs_${ts}.prof"
            } &
            
            # Block Profile (detect blocking operations)
            {
                curl -s "http://localhost:9080/debug/pprof/block" \
                    -o "$PPROF_DIR/block/block_${ts}.prof" 2>/dev/null
                echo "[PPROF] Block profile saved: block_${ts}.prof"
            } &
            
            sleep $PPROF_INTERVAL
        done
    ) &
    PPROF_PID=$!
    print_success "pprof collection started (PID: $PPROF_PID)"
}

# Redis monitoring with command-level analysis
start_redis_monitoring() {
    print_forensic "Starting Redis command monitoring..."
    
    (
        # Redis MONITOR for real-time command analysis
        docker exec insight-collector-redis redis-cli monitor > "$TRACES_DIR/redis/redis_commands.log" 2>&1 &
        local monitor_pid=$!
        
        # Redis latency tracking
        while true; do
            local ts=$(date +"%Y-%m-%d %H:%M:%S")
            {
                echo "[$ts] Redis Latency Analysis:"
                docker exec insight-collector-redis redis-cli --latency-history -i 1 -c 5 2>/dev/null | tail -5
                echo "---"
            } >> "$TRACES_DIR/redis/redis_latency.log"
            sleep 30
        done &
        
        wait $monitor_pid
    ) &
    REDIS_MONITOR_PID=$!
    print_success "Redis monitoring started (PID: $REDIS_MONITOR_PID)"
}

# Enhanced memory analysis
get_memory_forensics() {
    local container_name="insight-collector"
    
    echo -n "📊 Memory Details: "
    local mem_info=$(docker exec $container_name cat /proc/self/status 2>/dev/null | grep -E "(VmPeak|VmSize|VmLck|VmPin|VmHWM|VmRSS|VmData|VmStk|VmExe|VmLib|VmSwap)" | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g' || echo "N/A")
    echo "$mem_info"
    
    echo -n "🧠 Memory Pressure: "
    local mem_pressure=$(docker exec $container_name cat /proc/pressure/memory 2>/dev/null | head -1 || echo "N/A")
    echo "$mem_pressure"
}

# Deep container resource analysis  
get_container_forensics() {
    local container_name="insight-collector"
    
    echo -n "🖥️  Container Resources: "
    local container_stats=$(docker stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" $container_name 2>/dev/null | tail -1 | tr '\t' ' ' || echo "N/A")
    echo "$container_stats"
    
    echo -n "💾 Container Limits: "
    local limits=$(docker exec $container_name sh -c 'echo "CPU: $(nproc) cores, MEM: $(free -h | head -2 | tail -1 | awk "{print \$2}"), SWAP: $(free -h | tail -1 | awk "{print \$2}")"' 2>/dev/null || echo "N/A")
    echo "$limits"
}

# Advanced Go runtime analysis
get_go_runtime_forensics() {
    local debug_url="http://localhost:9080/debug/vars"
    
    echo -n "🔧 Go Runtime Deep: "
    local runtime_detailed=$(curl -s "$debug_url" 2>/dev/null | jq -c '{
        Goroutines: (.NumGoroutine // 0),
        CGoCalls: (.NumCgoCall // 0),
        Memory: {
            Alloc: (.memstats.Alloc // 0),
            TotalAlloc: (.memstats.TotalAlloc // 0),
            Sys: (.memstats.Sys // 0),
            Lookups: (.memstats.Lookups // 0),
            Mallocs: (.memstats.Mallocs // 0),
            Frees: (.memstats.Frees // 0),
            HeapAlloc: (.memstats.HeapAlloc // 0),
            HeapSys: (.memstats.HeapSys // 0),
            HeapIdle: (.memstats.HeapIdle // 0),
            HeapInuse: (.memstats.HeapInuse // 0),
            HeapReleased: (.memstats.HeapReleased // 0),
            HeapObjects: (.memstats.HeapObjects // 0),
            StackInuse: (.memstats.StackInuse // 0),
            StackSys: (.memstats.StackSys // 0),
            MSpanInuse: (.memstats.MSpanInuse // 0),
            MSpanSys: (.memstats.MSpanSys // 0),
            MCacheInuse: (.memstats.MCacheInuse // 0),
            MCacheSys: (.memstats.MCacheSys // 0),
            NextGC: (.memstats.NextGC // 0),
            NumGC: (.memstats.NumGC // 0),
            GCCPUFraction: (.memstats.GCCPUFraction // 0)
        }
    }' || echo '{"error": "runtime_unavailable"}')
    echo "$runtime_detailed"
    
    echo -n "⚙️  GC Stats: "
    local gc_stats=$(curl -s "$debug_url" 2>/dev/null | jq -c '{
        LastGC: (.memstats.LastGC // 0),
        PauseTotalNs: (.memstats.PauseTotalNs // 0),
        NumGC: (.memstats.NumGC // 0),
        NumForcedGC: (.memstats.NumForcedGC // 0),
        GCCPUFraction: (.memstats.GCCPUFraction // 0)
    }' || echo '{"error": "gc_unavailable"}')
    echo "$gc_stats"
}

# HTTP connection deep analysis
get_http_forensics() {
    local container_name="insight-collector"
    
    echo -n "🌐 HTTP Connections: "
    local total_connections=$(docker exec $container_name ss -tuln 2>/dev/null | grep -c ":8080" || echo "0")
    echo "Total: $total_connections"
    
    echo -n "🔗 TCP States Detail: "
    local tcp_states=$(docker exec $container_name ss -antup 2>/dev/null | grep ":8080" | awk '{print $1}' | sort | uniq -c | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g' || echo "N/A")
    echo "$tcp_states"
    
    echo -n "📡 Network Stats: "
    local net_stats=$(docker exec $container_name cat /proc/net/dev 2>/dev/null | grep eth0 | awk '{print "RX_bytes:"$2" TX_bytes:"$10" RX_packets:"$3" TX_packets:"$11}' || echo "N/A")
    echo "$net_stats"
    
    # File descriptor analysis
    echo -n "📂 File Descriptors: "
    local fd_info=$(docker exec $container_name sh -c 'echo "Open: $(ls /proc/self/fd | wc -l), Limit: $(ulimit -n)"' 2>/dev/null || echo "N/A")
    echo "$fd_info"
}

# Redis deep forensics
get_redis_forensics() {
    echo -n "🔍 Redis Commands/sec: "
    local ops_per_sec=$(docker exec insight-collector-redis redis-cli info stats 2>/dev/null | grep "instantaneous_ops_per_sec" | cut -d: -f2 | tr -d '\r\n' || echo "0")
    echo "$ops_per_sec"
    
    echo -n "💾 Redis Memory Detail: "
    local redis_mem=$(docker exec insight-collector-redis redis-cli info memory 2>/dev/null | grep -E "(used_memory:|used_memory_human:|used_memory_peak:|used_memory_peak_human:)" | tr '\n' ' ' | tr -d '\r' || echo "N/A")
    echo "$redis_mem"
    
    echo -n "🔄 Redis Connections: "
    local redis_conn=$(docker exec insight-collector-redis redis-cli info clients 2>/dev/null | grep -E "(connected_clients:|client_recent_max_input_buffer:|client_recent_max_output_buffer:)" | tr '\n' ' ' | tr -d '\r' || echo "N/A")
    echo "$redis_conn"
    
    echo -n "⚡ Redis Performance: "
    local redis_perf=$(docker exec insight-collector-redis redis-cli info stats 2>/dev/null | grep -E "(total_commands_processed:|total_connections_received:|keyspace_hits:|keyspace_misses:)" | tr '\n' ' ' | tr -d '\r' || echo "N/A")
    echo "$redis_perf"
}

# Comprehensive Asynq analysis
get_asynq_forensics() {
    echo -n "📋 Asynq Queues Deep: "
    # Get detailed queue information
    local critical_pending=$(docker exec insight-collector-redis redis-cli llen "asynq:{critical}:pending" 2>/dev/null || echo "0")
    local critical_active=$(docker exec insight-collector-redis redis-cli llen "asynq:{critical}:active" 2>/dev/null || echo "0") 
    local critical_retry=$(docker exec insight-collector-redis redis-cli zcard "asynq:{critical}:retry" 2>/dev/null || echo "0")
    local critical_dead=$(docker exec insight-collector-redis redis-cli zcard "asynq:{critical}:dead" 2>/dev/null || echo "0")
    local low_pending=$(docker exec insight-collector-redis redis-cli llen "asynq:{low}:pending" 2>/dev/null || echo "0")
    local low_active=$(docker exec insight-collector-redis redis-cli llen "asynq:{low}:active" 2>/dev/null || echo "0")
    echo "Critical(P:$critical_pending A:$critical_active R:$critical_retry D:$critical_dead) Low(P:$low_pending A:$low_active)"
    
    echo -n "👷 Asynq Workers: "
    local workers_info=$(docker exec insight-collector-redis redis-cli hgetall "asynq:workers" 2>/dev/null | paste - - | wc -l || echo "0")
    echo "Count: $workers_info"
    
    echo -n "📊 Asynq Stats Deep: "
    local asynq_stats=$(docker exec insight-collector-redis redis-cli hgetall "asynq:stats" 2>/dev/null | paste - - | head -5 | tr '\n' ' ' || echo "N/A")
    echo "$asynq_stats"
    
    # Get Asynq server information
    echo -n "🖥️  Asynq Servers: "
    local servers_count=$(docker exec insight-collector-redis redis-cli keys "asynq:servers:*" 2>/dev/null | wc -l || echo "0")
    echo "Active: $servers_count"
}

# HTTP request timing with detailed breakdown
get_request_timing_forensics() {
    echo -n "⏱️  Health Endpoint Timing: "
    local health_timing=$(curl -w "@-" -o /dev/null -s "http://localhost:8080/v1/health/live" <<'EOF'
{"dns":%{time_namelookup},"connect":%{time_connect},"appconnect":%{time_appconnect},"pretransfer":%{time_pretransfer},"starttransfer":%{time_starttransfer},"total":%{time_total},"size_download":%{size_download},"speed_download":%{speed_download}}
EOF
)
    echo "$health_timing" | jq -c '.' 2>/dev/null || echo "$health_timing"
    
    # Test Asynq endpoint timing
    echo -n "⏱️  Asynq Endpoint Timing: "
    local asynq_payload='{"user_id":"forensic_test","session_id":"forensic_session","activity_type":"forensic_analysis","status":"testing","channel":"load_test","duration_ms":100,"response_code":200,"request_id":"forensic_req","trace_id":"forensic_trace","ip_address":"127.0.0.1","user_agent":"ForensicAnalyzer/1.0","_test_metadata":{"forensic":true}}'
    local asynq_timing=$(curl -w "@-" -o /dev/null -s -X POST "http://localhost:8080/v1/user-activities/insert" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer dummy-token" \
        -d "$asynq_payload" <<'EOF'
{"dns":%{time_namelookup},"connect":%{time_connect},"appconnect":%{time_appconnect},"pretransfer":%{time_pretransfer},"starttransfer":%{time_starttransfer},"total":%{time_total},"http_code":%{http_code}}
EOF
)
    echo "$asynq_timing" | jq -c '.' 2>/dev/null || echo "$asynq_timing"
}

# System pressure and resource contention
get_system_forensics() {
    local container_name="insight-collector"
    
    echo -n "🚨 Load Average: "
    local load_avg=$(docker exec $container_name cat /proc/loadavg 2>/dev/null || echo "N/A")
    echo "$load_avg"
    
    echo -n "⚡ CPU Pressure: "
    local cpu_pressure=$(docker exec $container_name cat /proc/pressure/cpu 2>/dev/null | head -1 || echo "N/A")
    echo "$cpu_pressure"
    
    echo -n "💽 IO Pressure: "
    local io_pressure=$(docker exec $container_name cat /proc/pressure/io 2>/dev/null | head -1 || echo "N/A")
    echo "$io_pressure"
    
    echo -n "🔧 System Limits: "
    local limits=$(docker exec $container_name sh -c 'echo "Files: $(ulimit -n), Processes: $(ulimit -u), Memory: $(ulimit -v)"' 2>/dev/null || echo "N/A")
    echo "$limits"
    
    echo -n "📈 Context Switches: "
    local ctx_switches=$(docker exec $container_name grep -E "(ctxt|processes)" /proc/stat 2>/dev/null | tr '\n' ' ' || echo "N/A")
    echo "$ctx_switches"
}

# Application-specific debug information
get_application_debug() {
    echo -n "🔧 Application Debug: "
    local debug_info=$(curl -s "http://localhost:8080/debug/connections" 2>/dev/null | jq -c '.' || echo '{"error": "debug_unavailable"}')
    echo "$debug_info"
    
    echo -n "📈 Application Health: "
    local health_info=$(curl -s "http://localhost:8080/v1/health" 2>/dev/null | jq -c '{success: .success, data: {influxdb: .data.influxdb, redis: .data.redis}}' || echo '{"error": "health_unavailable"}')
    echo "$health_info"
}

# Trace data collection for deep analysis
collect_trace_data() {
    local check_num=$1
    local ts=$(date +"%H%M%S")
    
    # Save detailed snapshots every minute
    if (( check_num % 6 == 0 )); then  # Every 6 checks = every minute
        print_trace "Collecting trace data snapshot at $ts..."
        
        # Redis state snapshot
        {
            echo "=== Redis State Snapshot at $(date) ==="
            docker exec insight-collector-redis redis-cli info all 2>/dev/null || echo "Redis info unavailable"
            echo "=== Asynq Keys ==="
            docker exec insight-collector-redis redis-cli keys "asynq:*" 2>/dev/null || echo "Asynq keys unavailable"
        } >> "$TRACES_DIR/redis/redis_state_${ts}.log"
        
        # Go runtime snapshot
        {
            echo "=== Go Runtime Snapshot at $(date) ==="
            curl -s "http://localhost:9080/debug/vars" 2>/dev/null | jq '.' || echo "Runtime vars unavailable"
        } >> "$TRACES_DIR/system/runtime_${ts}.json"
        
        # HTTP connections snapshot
        {
            echo "=== HTTP Connections Snapshot at $(date) ==="
            docker exec insight-collector ss -antup 2>/dev/null | grep ":8080" || echo "No HTTP connections"
        } >> "$TRACES_DIR/http/connections_${ts}.log"
    fi
}

# Main enhanced monitoring function
run_enhanced_forensic_check() {
    local check_num=$1
    local elapsed=$((check_num * CHECK_INTERVAL))
    local remaining=$((MONITORING_DURATION - elapsed))
    
    print_info "$(date +%H:%M:%S) === FORENSIC CHECK $check_num/$TOTAL_CHECKS (${elapsed}s elapsed, ${remaining}s remaining) ==="
    
    # Basic system metrics
    get_memory_forensics
    get_container_forensics
    get_go_runtime_forensics
    
    # Network and HTTP analysis
    get_http_forensics
    
    # Redis and Asynq deep dive
    get_redis_forensics
    get_asynq_forensics
    
    # Request timing analysis
    get_request_timing_forensics
    
    # System pressure analysis
    get_system_forensics
    
    # Application debug
    get_application_debug
    
    # Collect trace data for analysis
    collect_trace_data $check_num
    
    echo ""
}

# Start background monitoring
print_forensic "Initializing background monitoring processes..."
start_pprof_collection
start_redis_monitoring
sleep 2

print_success "All monitoring processes started successfully!"
print_info "Starting enhanced forensic monitoring loop..."
echo ""

# Main monitoring loop
for ((i=1; i<=TOTAL_CHECKS; i++)); do
    run_enhanced_forensic_check $i
    
    # Don't sleep after the last check
    if [[ $i -lt $TOTAL_CHECKS ]]; then
        sleep $CHECK_INTERVAL
    fi
done

print_success "Enhanced forensic monitoring completed!"
print_forensic "Final report:"
echo "  • Total monitoring duration: ${MONITORING_DURATION}s"
echo "  • Total checks performed: $TOTAL_CHECKS"
echo "  • Check interval: ${CHECK_INTERVAL}s"
echo "  • pprof profiles collected: ~$((MONITORING_DURATION / PPROF_INTERVAL))"
echo "  • Detailed log: $LOG_FILE"
echo "  • Profiles directory: $PPROF_DIR/"
echo "  • Traces directory: $TRACES_DIR/"
echo ""
print_forensic "Data ready for comprehensive Asynq bottleneck analysis!"

# Clean up background processes
cleanup_on_exit