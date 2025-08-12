#!/bin/bash

# Single Endpoint Testing Script for InsightCollector
# Usage: ./single-endpoint-test.sh [endpoint] [host] [options]

set -e

# Configuration
DEFAULT_HOST="http://host.docker.internal:8080"
DEFAULT_JWT="dummy-token-for-non-auth-testing"
RESULTS_DIR="results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Valid endpoints
VALID_ENDPOINTS=("user-activities" "transaction-events" "security-events" "callback-logs")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}🎯 $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

show_usage() {
    echo "🎯 Single Endpoint Testing for InsightCollector"
    echo ""
    echo "Usage: $0 [ENDPOINT] [HOST] [OPTIONS]"
    echo ""
    echo "ENDPOINTS:"
    echo "  user-activities       - User interaction logging (default)"
    echo "  transaction-events    - Financial transaction tracking"
    echo "  security-events       - Security incident monitoring"
    echo "  callback-logs         - External webhook processing"
    echo "  all                   - Test all endpoints sequentially"
    echo "  compare               - Test all and generate comparison"
    echo ""
    echo "OPTIONS:"
    echo "  --with-metrics        Enable metrics collection"
    echo "  --analyze             Run post-test analysis"
    echo "  --intensive           Use intensive test patterns"
    echo ""
    echo "Examples:"
    echo "  $0 user-activities                           # Test user-activities only"
    echo "  $0 transaction-events --with-metrics         # Test with metrics"
    echo "  $0 all                                       # Test all endpoints"
    echo "  $0 compare --analyze                         # Test all + comparison"
    echo "  $0 security-events http://localhost:8080     # Custom host"
    echo ""
    exit 0
}

# Function to validate endpoint
validate_endpoint() {
    local endpoint=$1
    
    if [[ " ${VALID_ENDPOINTS[@]} " =~ " ${endpoint} " ]] || [[ "$endpoint" == "all" ]] || [[ "$endpoint" == "compare" ]]; then
        return 0
    else
        print_error "Invalid endpoint: $endpoint"
        print_info "Valid endpoints: ${VALID_ENDPOINTS[*]} all compare"
        exit 1
    fi
}

# Function to check Docker Compose
check_docker_compose() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose is not available"
        exit 1
    fi
}

# Function to run single endpoint test
run_single_endpoint_test() {
    local endpoint=$1
    local with_metrics=${2:-false}
    
    print_info "Testing single endpoint: $endpoint"
    
    # Prepare environment
    cat > .env << EOF
TEST_HOST=$HOST
JWT_TOKEN=$JWT_TOKEN
TARGET_ENDPOINT=$endpoint
TIMESTAMP=$TIMESTAMP
EOF
    
    # Build command
    local json_output="/app/results/single_${endpoint}_${TIMESTAMP}.json"
    local summary_output="/app/results/single_${endpoint}_${TIMESTAMP}_summary.txt"
    
    local k6_cmd="run"
    k6_cmd="$k6_cmd --out json=$json_output"
    k6_cmd="$k6_cmd --summary-export=$summary_output"
    
    # Add metrics output if enabled
    if [[ "$with_metrics" == "true" ]]; then
        k6_cmd="$k6_cmd --out influxdb=http://k6-influxdb:8086/k6-metrics"
    fi
    
    k6_cmd="$k6_cmd scenarios/single-endpoint.js"
    
    # Run test
    $DOCKER_COMPOSE_CMD run --rm \
        -e TEST_HOST="$HOST" \
        -e JWT_TOKEN="$JWT_TOKEN" \
        -e TARGET_ENDPOINT="$endpoint" \
        k6 $k6_cmd 2>&1 | tee "results/single_${endpoint}_${TIMESTAMP}_console.log"
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        print_success "✅ $endpoint test completed"
        
        # Show quick summary
        if [ -f "results/single_${endpoint}_${TIMESTAMP}_summary.txt" ]; then
            echo ""
            echo "📊 $endpoint Quick Results:"
            grep -E "(http_req_duration|${endpoint//-/_}.*duration)" "results/single_${endpoint}_${TIMESTAMP}_summary.txt" | head -3 | sed 's/^/  /'
            echo ""
        fi
        
        return 0
    else
        print_error "❌ $endpoint test failed"
        return $exit_code
    fi
}

# Function to run all single endpoint tests
run_all_endpoints() {
    local with_metrics=${1:-false}
    local results=()
    
    print_info "Running all single endpoint tests..."
    
    for endpoint in "${VALID_ENDPOINTS[@]}"; do
        print_info "Starting test: $endpoint"
        
        if run_single_endpoint_test "$endpoint" "$with_metrics"; then
            results+=("$endpoint:SUCCESS")
        else
            results+=("$endpoint:FAILED")
        fi
        
        # Brief pause between tests
        sleep 3
    done
    
    # Show summary
    echo ""
    print_info "📊 All Endpoint Tests Summary:"
    for result in "${results[@]}"; do
        IFS=':' read -r endpoint status <<< "$result"
        if [[ "$status" == "SUCCESS" ]]; then
            print_success "$endpoint: PASSED"
        else
            print_error "$endpoint: FAILED"
        fi
    done
    echo ""
}

# Function to run comparison test
run_comparison_test() {
    local with_metrics=${1:-false}
    
    print_info "Running endpoint comparison test..."
    
    # Run all endpoints
    run_all_endpoints "$with_metrics"
    
    # Generate comparison report
    print_info "Generating endpoint comparison report..."
    
    local report_file="results/endpoint_comparison_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# Endpoint Performance Comparison - $(date)

## Test Configuration
- **Host**: $HOST
- **Timestamp**: $TIMESTAMP
- **Test Mode**: Single Endpoint Intensive

## Performance Summary

| Endpoint | Avg Response | 95th Percentile | Error Rate | Status |
|----------|-------------|-----------------|------------|---------|
EOF

    # Add data for each endpoint
    for endpoint in "${VALID_ENDPOINTS[@]}"; do
        local summary_file="results/single_${endpoint}_${TIMESTAMP}_summary.txt"
        
        if [ -f "$summary_file" ]; then
            # Extract metrics (simplified parsing)
            local avg_duration="N/A"
            local p95_duration="N/A" 
            local error_rate="N/A"
            local status="❌ No Data"
            
            if grep -q "http_req_duration" "$summary_file"; then
                avg_duration=$(grep "http_req_duration" "$summary_file" | grep -o "avg=[0-9.]*[a-z]*" | cut -d= -f2 || echo "N/A")
                p95_duration=$(grep "http_req_duration" "$summary_file" | grep -o "p(95)=[0-9.]*[a-z]*" | cut -d= -f2 || echo "N/A")
                error_rate=$(grep "http_req_failed" "$summary_file" | grep -o "[0-9.]*%" | head -1 || echo "N/A")
                status="✅ Completed"
            fi
            
            echo "| $endpoint | $avg_duration | $p95_duration | $error_rate | $status |" >> "$report_file"
        else
            echo "| $endpoint | N/A | N/A | N/A | ❌ Failed |" >> "$report_file"
        fi
    done
    
    # Add recommendations
    cat >> "$report_file" << EOF

## Performance Analysis

### Response Time Ranking
$(for endpoint in "${VALID_ENDPOINTS[@]}"; do
    summary_file="results/single_${endpoint}_${TIMESTAMP}_summary.txt"
    if [ -f "$summary_file" ] && grep -q "http_req_duration" "$summary_file"; then
        avg=$(grep "http_req_duration" "$summary_file" | grep -o "avg=[0-9.]*" | cut -d= -f2 | head -1)
        echo "- $endpoint: ${avg}ms avg"
    fi
done | sort -k2 -n)

### Recommendations
- **Fastest**: Best optimized endpoint
- **Slowest**: May need optimization
- **Error Prone**: Check endpoints with >2% error rate
- **Scaling**: Consider worker allocation based on response times

## Files Generated
$(ls -1 results/single_*_${TIMESTAMP}* | sort)

---
Generated by Single Endpoint Testing Suite
EOF
    
    print_success "Comparison report generated: $report_file"
    
    # Show quick comparison
    echo ""
    print_info "📊 Quick Performance Comparison:"
    for endpoint in "${VALID_ENDPOINTS[@]}"; do
        local summary_file="results/single_${endpoint}_${TIMESTAMP}_summary.txt"
        if [ -f "$summary_file" ] && grep -q "http_req_duration" "$summary_file"; then
            local avg=$(grep "http_req_duration" "$summary_file" | grep -o "avg=[0-9.]*[a-z]*" | cut -d= -f2)
            local p95=$(grep "http_req_duration" "$summary_file" | grep -o "p(95)=[0-9.]*[a-z]*" | cut -d= -f2)
            echo "  • $endpoint: $avg avg, $p95 p95"
        fi
    done
    echo ""
}

# Main execution
main() {
    ENDPOINT=${1:-"user-activities"}
    HOST=${2:-$DEFAULT_HOST}
    JWT_TOKEN=${3:-$DEFAULT_JWT}
    
    # Show usage if requested
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        show_usage
    fi
    
    # Parse options
    WITH_METRICS=false
    RUN_ANALYSIS=false
    INTENSIVE_MODE=false
    
    for arg in "$@"; do
        case $arg in
            --with-metrics)
                WITH_METRICS=true
                ;;
            --analyze)
                RUN_ANALYSIS=true
                ;;
            --intensive)
                INTENSIVE_MODE=true
                ;;
        esac
    done
    
    # Validate endpoint
    validate_endpoint "$ENDPOINT"
    
    print_info "Single Endpoint Test Configuration:"
    echo "  • Target Endpoint: $ENDPOINT"
    echo "  • Target Host: $HOST"
    echo "  • JWT Token: dummy-token (non-auth testing)"
    echo "  • With Metrics: $WITH_METRICS"
    echo "  • Run Analysis: $RUN_ANALYSIS"
    echo "  • Intensive Mode: $INTENSIVE_MODE"
    echo ""
    
    # Check prerequisites
    check_docker_compose
    mkdir -p "$RESULTS_DIR"
    
    # Health check
    print_info "Health checking target service..."
    local check_url=$(echo "$HOST" | sed 's/host\.docker\.internal/localhost/')
    if ! curl -f -s --max-time 10 "$check_url/v1/health/live" > /dev/null; then
        print_error "Health check failed - service not available"
        exit 1
    fi
    print_success "Service is ready"
    
    # Start metrics if requested
    if [[ "$WITH_METRICS" == "true" ]]; then
        print_info "Starting metrics services..."
        docker compose --profile metrics up -d k6-influxdb k6-grafana
        sleep 10
    fi
    
    # Run tests based on endpoint selection
    case $ENDPOINT in
        "all")
            run_all_endpoints "$WITH_METRICS"
            ;;
        "compare")
            run_comparison_test "$WITH_METRICS"
            ;;
        *)
            run_single_endpoint_test "$ENDPOINT" "$WITH_METRICS"
            ;;
    esac
    
    # Run analysis if requested
    if [[ "$RUN_ANALYSIS" == "true" ]]; then
        print_info "Running analysis..."
        # Add analysis logic here
    fi
    
    print_success "Single endpoint testing completed! 🎯"
    print_info "Check results in the 'results/' directory"
    
    if [[ "$WITH_METRICS" == "true" ]]; then
        echo ""
        print_info "Metrics dashboards available:"
        echo "  • Grafana: http://localhost:3001 (admin/k6admin)"
        echo "  • InfluxDB: http://localhost:8087 (k6/k6password)"
    fi
}

# Run main function
main "$@"