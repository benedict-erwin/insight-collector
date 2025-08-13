#!/bin/bash

# Docker-based K6 Load Testing Runner for InsightCollector
# Usage: ./docker-run.sh [test-type] [host] [options]

set -e

# Signal handler for graceful shutdown
cleanup_on_exit() {
    local exit_code=$?
    print_warning "Received interrupt signal. Performing cleanup..."
    
    # Determine docker compose command if not set
    if [[ -z "$DOCKER_COMPOSE_CMD" ]]; then
        if docker compose version &> /dev/null; then
            DOCKER_COMPOSE_CMD="docker compose"
        elif docker-compose --version &> /dev/null; then
            DOCKER_COMPOSE_CMD="docker-compose"
        else
            print_error "Docker Compose not found. Manual cleanup required."
            exit $exit_code
        fi
    fi
    
    # Stop K6 test if running
    print_info "Stopping K6 test containers..."
    $DOCKER_COMPOSE_CMD stop k6 2>/dev/null || true
    
    # If metrics were started, ask user if they want to keep them running
    if [[ "$WITH_METRICS" == "true" ]]; then
        echo ""
        print_info "Metrics services (InfluxDB & Grafana) are still running."
        read -p "Do you want to stop metrics services too? [y/N]: " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "Stopping metrics services..."
            $DOCKER_COMPOSE_CMD --profile metrics down
            print_success "All services stopped"
        else
            print_success "Metrics services still running:"
            echo "  • Grafana: http://localhost:3001 (admin/k6admin)"
            echo "  • InfluxDB v1.8: http://localhost:8087 (k6/k6password)"
            echo "  • To stop later: $DOCKER_COMPOSE_CMD --profile metrics down"
        fi
    else
        # If no metrics, just stop any running containers
        print_info "Stopping any running containers..."
        $DOCKER_COMPOSE_CMD down 2>/dev/null || true
    fi
    
    print_info "Cleanup completed"
    exit $exit_code
}

# Configuration
DEFAULT_HOST="http://host.docker.internal:8080"  # Docker host networking
DEFAULT_JWT="dummy-token-for-non-auth-testing"    # Non-auth testing
RESULTS_DIR="results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}🐳 $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# Set trap for SIGINT (Ctrl+C) and SIGTERM
trap cleanup_on_exit SIGINT SIGTERM

show_usage() {
    echo "🐳 Docker-based K6 Load Testing for InsightCollector"
    echo ""
    echo "Usage: $0 [TEST_TYPE] [HOST] [OPTIONS]"
    echo ""
    echo "TEST_TYPES:"
    echo "  smoke         - Quick smoke test (30s, 1 user)"
    echo "  load          - Normal load test (9m, up to 10 users)"
    echo "  stress        - Stress test (16m, up to 50 users)"  
    echo "  spike         - Spike test (1m, spike to 100 users)"
    echo "  mixed         - Mixed endpoint test (all endpoints together)"
    echo "  endpoint      - Individual endpoint focus test (all endpoints separately)"
    echo "  single        - Single endpoint intensive test"
    echo "  health-only   - Health endpoint stress test (16m, up to 50 users)"
    echo "  asynq-only    - Asynq endpoint stress test (16m, up to 50 users)"
    echo "  all           - Run all test types sequentially"
    echo ""
    echo "OPTIONS:"
    echo "  --with-metrics    Start InfluxDB + Grafana for metrics collection"
    echo "  --analyze         Run post-test analysis"
    echo "  --cleanup         Clean up containers and volumes"
    echo "  --logs            Show K6 container logs"
    echo ""
    echo "Examples:"
    echo "  $0 smoke                                    # Basic smoke test"
    echo "  $0 load http://localhost:8080              # Custom host"
    echo "  $0 stress --with-metrics                   # With metrics collection"
    echo "  $0 mixed http://host.docker.internal:8080  # Test all endpoints together"
    echo "  $0 single user-activities                  # Test single endpoint"
    echo "  $0 endpoint                                # Test all endpoints separately"
    echo "  $0 --cleanup                               # Clean up everything"
    echo ""
    echo "💡 For non-auth testing, JWT token is set to dummy value automatically"
    exit 0
}

# Function to check Docker
check_docker() {
    print_info "Checking Docker availability..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    fi
    
    print_success "Docker is available and running"
    
    # Check if docker-compose is available (v2 or v1)
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker compose"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose is not available"
        exit 1
    fi
    
    print_success "Using: $DOCKER_COMPOSE_CMD"
}

# Function to prepare environment
prepare_environment() {
    print_info "Preparing test environment..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Create .env file for docker-compose
    cat > .env << EOF
TEST_HOST=$TEST_HOST
JWT_TOKEN=$JWT_TOKEN
TEST_TYPE=$TEST_TYPE
TIMESTAMP=$TIMESTAMP
EOF
    
    print_success "Environment prepared"
}

# Function to health check target service
health_check() {
    print_info "Health checking target service: $TEST_HOST"
    
    # Use a temporary container to check health
    local health_url="$TEST_HOST/v1/health/live"
    
    # Convert host.docker.internal to localhost for health check from host
    local check_url="$health_url"
    if [[ "$health_url" == *"host.docker.internal"* ]]; then
        check_url=$(echo "$health_url" | sed 's/host\.docker\.internal/localhost/')
    fi
    
    if curl -f -s --max-time 10 "$check_url" > /dev/null; then
        print_success "Health check passed - service is ready"
    else
        print_error "Health check failed - service may not be available at $check_url"
        print_info "Make sure your InsightCollector service is running"
        print_info "For Docker: Use 'host.docker.internal:8080' (default)"  
        print_info "For local: Use 'localhost:8080'"
        exit 1
    fi
}

# Function to start metrics collection
start_metrics() {
    print_info "Starting metrics collection services..."
    
    $DOCKER_COMPOSE_CMD --profile metrics up -d k6-influxdb k6-grafana
    
    print_success "Metrics services started:"
    echo "  • InfluxDB v1.8: http://localhost:8087 (k6/k6password)"
    echo "  • Grafana: http://localhost:3001 (admin/k6admin)"
    
    print_info "Waiting for services to be ready..."
    sleep 15
}

# Function to run specific K6 test
run_k6_test() {
    local test_type=$1
    local script_name=$2
    
    print_info "Running K6 $test_type test in Docker container..."
    
    # Determine the script to run
    if [[ "$test_type" == "mixed" ]]; then
        script_name="scenarios/mixed-load.js"
    elif [[ "$test_type" == "endpoint" ]]; then
        script_name="scenarios/endpoint-specific.js"
    elif [[ "$test_type" == "health-only" ]]; then
        script_name="scenarios/health-only-isolated.js"
    elif [[ "$test_type" == "asynq-only" ]]; then
        script_name="scenarios/asynq-only.js"
    else
        script_name="scenarios/mixed-load.js"
    fi
    
    # Prepare output file paths
    local json_output="/app/results/${test_type}_${TIMESTAMP}.json"
    local summary_output="/app/results/${test_type}_${TIMESTAMP}_summary.txt"
    
    # Build K6 command
    local k6_cmd="run"
    k6_cmd="$k6_cmd --env TEST_HOST=$TEST_HOST"
    k6_cmd="$k6_cmd --env JWT_TOKEN=$JWT_TOKEN"
    k6_cmd="$k6_cmd --env TEST_TYPE=$test_type"
    k6_cmd="$k6_cmd --out json=$json_output"
    k6_cmd="$k6_cmd --summary-export=$summary_output"
    
     # Add InfluxDB output ONLY if explicitly requested with --with-metrics
    if [[ "$WITH_METRICS" == "true" ]] && docker ps --format "table {{.Names}}" | grep -q "insight-k6-influxdb"; then
        k6_cmd="$k6_cmd --out influxdb=http://k6-influxdb:8086/k6?pushInterval=5s"
        k6_cmd="$k6_cmd --tag testid=${test_type}_${TIMESTAMP}"
    fi
    
    k6_cmd="$k6_cmd $script_name"
    
    # Run K6 test
    $DOCKER_COMPOSE_CMD run --rm \
        -e TEST_HOST="$TEST_HOST" \
        -e JWT_TOKEN="$JWT_TOKEN" \
        -e TEST_TYPE="$test_type" \
        k6 $k6_cmd
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        print_success "$test_type test completed successfully"
        print_info "Results saved to:"
        echo "  • JSON: results/${test_type}_${TIMESTAMP}.json"
        echo "  • Summary: results/${test_type}_${TIMESTAMP}_summary.txt"
        
        # Show quick summary
        if [ -f "results/${test_type}_${TIMESTAMP}_summary.txt" ]; then
            echo ""
            echo "📊 Quick Summary:"
            grep -E "(http_req_duration|http_req_failed|data_received)" "results/${test_type}_${TIMESTAMP}_summary.txt" | head -5
        fi
    else
        print_error "$test_type test failed with exit code $exit_code"
        return $exit_code
    fi
}

# Function to run analysis
run_analysis() {
    print_info "Running test results analysis..."
    
    # Build analyzer if it doesn't exist
    if [[ ! -f "analyzer/Dockerfile" ]]; then
        print_info "Creating simple analysis script..."
        mkdir -p analyzer
        cat > analyzer/analyze.py << 'EOF'
#!/usr/bin/env python3
import json
import glob
import os
from datetime import datetime

def analyze_results():
    results_dir = os.environ.get('RESULTS_DIR', '/app/results')
    json_files = glob.glob(f"{results_dir}/*_*.json")
    
    if not json_files:
        print("No JSON result files found")
        return
    
    print("🔍 K6 Test Results Analysis")
    print("=" * 40)
    
    for json_file in sorted(json_files):
        print(f"\n📊 Analyzing: {os.path.basename(json_file)}")
        
        try:
            with open(json_file, 'r') as f:
                lines = f.readlines()
                
            # Count metrics
            total_requests = len([l for l in lines if '"type":"Point"' in l and '"metric":"http_reqs"' in l])
            error_lines = [l for l in lines if '"type":"Point"' in l and '"metric":"http_req_failed"' in l and '"value":1' in l]
            error_count = len(error_lines)
            
            print(f"  • Total Requests: {total_requests}")
            print(f"  • Failed Requests: {error_count}")
            if total_requests > 0:
                error_rate = (error_count / total_requests) * 100
                print(f"  • Error Rate: {error_rate:.2f}%")
                
                if error_rate < 2:
                    print("  • Status: ✅ GOOD")
                elif error_rate < 5:
                    print("  • Status: ⚠️  WARNING") 
                else:
                    print("  • Status: ❌ POOR")
            
        except Exception as e:
            print(f"  • Error analyzing file: {e}")

if __name__ == "__main__":
    analyze_results()
EOF
        
        cat > analyzer/Dockerfile << 'EOF'
FROM python:3.9-alpine
WORKDIR /app
COPY analyze.py /app/
RUN chmod +x /app/analyze.py
CMD ["python3", "/app/analyze.py"]
EOF
    fi
    
    # Run analysis
    $DOCKER_COMPOSE_CMD --profile analysis build k6-analyzer
    $DOCKER_COMPOSE_CMD --profile analysis run --rm k6-analyzer
}

# Function to show logs
show_logs() {
    print_info "Showing K6 container logs..."
    $DOCKER_COMPOSE_CMD logs k6
}

# Function for full cleanup
full_cleanup() {
    print_info "Cleaning up containers and volumes..."

    # Detect docker compose command if not already set
    if [[ -z "$DOCKER_COMPOSE_CMD" ]]; then
        if docker compose version &> /dev/null; then
            DOCKER_COMPOSE_CMD="docker compose"
        elif docker-compose --version &> /dev/null; then
            DOCKER_COMPOSE_CMD="docker-compose"
        else
            print_error "Docker Compose not found. Manual cleanup required."
            exit 1
        fi
    fi

    $DOCKER_COMPOSE_CMD --profile metrics --profile manual --profile analysis down -v
    
    # Remove custom networks
    docker network rm insight-k6-network 2>/dev/null || true
    
    # Remove custom volumes
    docker volume rm insight-k6-influxdb-data 2>/dev/null || true
    docker volume rm insight-k6-influxdb-config 2>/dev/null || true  
    docker volume rm insight-k6-grafana-data 2>/dev/null || true
    
    print_success "Cleanup completed"
}

# Main execution
  main() {
    # Set defaults
    TEST_TYPE="load"
    TEST_HOST=$DEFAULT_HOST
    JWT_TOKEN=$DEFAULT_JWT
    WITH_METRICS=false
    RUN_ANALYSIS=false

    # Parse arguments properly
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                ;;
            --cleanup)
                full_cleanup
                exit 0
                ;;
            --logs)
                show_logs
                exit 0
                ;;
            --with-metrics)
                WITH_METRICS=true
                shift
                ;;
            --analyze)
                RUN_ANALYSIS=true
                shift
                ;;
            smoke|load|stress|spike|mixed|endpoint|single|all|health-only|asynq-only)
                TEST_TYPE=$1
                shift
                ;;
            http://*)
                TEST_HOST=$1
                shift
                ;;
            *)
                # Assume it's JWT token if not recognized
                if [[ ${#1} -gt 50 ]]; then
                    JWT_TOKEN=$1
                fi
                shift
                ;;
        esac
    done
    
    # Parse options
    for arg in "$@"; do
        case $arg in
            --with-metrics)
                WITH_METRICS=true
                ;;
            --analyze)
                RUN_ANALYSIS=true
                ;;
        esac
    done
    
    print_info "Docker K6 Load Test Configuration:"
    echo "  • Test Type: $TEST_TYPE"
    echo "  • Target Host: $TEST_HOST"
    echo "  • JWT Token: dummy-token (non-auth testing)"
    echo "  • With Metrics: $WITH_METRICS"
    echo "  • Run Analysis: $RUN_ANALYSIS"
    echo ""
    
    # Check prerequisites
    check_docker
    prepare_environment
    health_check
    
    # Start metrics collection if requested
    if [[ "$WITH_METRICS" == "true" ]]; then
        start_metrics
    fi
    
    # Run tests
    case $TEST_TYPE in
        "smoke"|"load"|"stress"|"spike"|"mixed"|"endpoint"|"health-only"|"asynq-only")
            run_k6_test "$TEST_TYPE"
            ;;
        "all")
            print_info "Running all test types sequentially..."
            
            run_k6_test "smoke" && \
            run_k6_test "load" && \
            run_k6_test "stress" && \
            run_k6_test "mixed" && \
            run_k6_test "endpoint"
            
            if [ $? -eq 0 ]; then
                print_success "All tests completed successfully!"
            else
                print_error "Some tests failed. Check individual results."
                exit 1
            fi
            ;;
        *)
            print_error "Unknown test type: $TEST_TYPE"
            show_usage
            ;;
    esac
    
    # Run analysis if requested
    if [[ "$RUN_ANALYSIS" == "true" ]]; then
        run_analysis
    fi
    
    print_success "Docker K6 testing completed! 🎉"
    print_info "Results are available in the 'results/' directory"
    
    if [[ "$WITH_METRICS" == "true" ]]; then
        echo ""
        print_info "Metrics services are still running:"
        echo "  • Grafana: http://localhost:3001 (admin/k6admin)"
        echo "  • InfluxDB v1.8: http://localhost:8087 (k6/k6password)"
        echo ""
        echo "To stop metrics services: $DOCKER_COMPOSE_CMD --profile metrics down"
    fi
}

# Run main function
main "$@"