#!/bin/bash

# K6 Load Testing Runner for InsightCollector
# Usage: ./run-tests.sh [test-type] [host] [jwt-token]

set -e

# Configuration
DEFAULT_HOST="http://localhost:8080"
DEFAULT_JWT="your-jwt-token-here"
RESULTS_DIR="results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to show usage
show_usage() {
    echo "🚀 K6 Load Testing for InsightCollector"
    echo ""
    echo "Usage: $0 [TEST_TYPE] [HOST] [JWT_TOKEN]"
    echo ""
    echo "TEST_TYPES:"
    echo "  smoke         - Quick smoke test (30s, 1 user)"
    echo "  load          - Normal load test (9m, up to 10 users)"
    echo "  stress        - Stress test (16m, up to 50 users)"  
    echo "  spike         - Spike test (1m, spike to 100 users)"
    echo "  mixed         - Mixed endpoint test (custom duration)"
    echo "  endpoint      - Individual endpoint focus test"
    echo "  all           - Run all test types sequentially"
    echo ""
    echo "Examples:"
    echo "  $0 smoke"
    echo "  $0 load http://localhost:8080 eyJhbGciOiJSUzI1NiIs..."
    echo "  $0 stress https://api.insight.com \$JWT_TOKEN"
    echo "  $0 all"
    echo ""
    exit 0
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check if k6 is installed
    if ! command -v k6 &> /dev/null; then
        print_error "k6 is not installed. Please install k6 first:"
        echo "  macOS: brew install k6"
        echo "  Ubuntu: sudo apt-get install k6"
        echo "  Or download from: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
    
    # Check k6 version
    K6_VERSION=$(k6 version --short)
    print_success "k6 installed: $K6_VERSION"
    
    # Check if JWT token is provided and valid looking
    if [[ ${#JWT_TOKEN} -lt 100 ]]; then
        print_warning "JWT token looks short. Make sure you're using a valid JWT token."
        print_info "You can generate one using your auth system or use the example from CLAUDE.md"
    fi
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    print_success "Results directory ready: $RESULTS_DIR"
}

# Function to health check target service
health_check() {
    print_info "Performing health check on $HOST..."
    
    local health_url="$HOST/v1/health/live"
    
    if curl -f -s --max-time 10 "$health_url" > /dev/null; then
        print_success "Health check passed - service is ready"
    else
        print_error "Health check failed - service may not be available at $HOST"
        print_info "Make sure the service is running and accessible"
        exit 1
    fi
}

# Function to run a specific test
run_test() {
    local test_type=$1
    local script_name=$2
    local extra_env=""
    
    print_info "Running $test_type test..."
    
    # Set test-specific environment variables
    case $test_type in
        "endpoint")
            extra_env="--env TARGET_ENDPOINT=user-activities"
            ;;
    esac
    
    # Prepare output files
    local json_output="$RESULTS_DIR/${test_type}_${TIMESTAMP}.json"
    local summary_output="$RESULTS_DIR/${test_type}_${TIMESTAMP}_summary.txt"
    
    # Run k6 test
    k6 run \
        --env TEST_HOST="$HOST" \
        --env JWT_TOKEN="$JWT_TOKEN" \
        --env TEST_TYPE="$test_type" \
        $extra_env \
        --out json="$json_output" \
        --summary-export="$summary_output" \
        "$script_name" 2>&1 | tee "$RESULTS_DIR/${test_type}_${TIMESTAMP}_console.log"
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        print_success "$test_type test completed successfully"
        print_info "Results saved to:"
        echo "  • JSON: $json_output"
        echo "  • Summary: $summary_output"
        echo "  • Console: $RESULTS_DIR/${test_type}_${TIMESTAMP}_console.log"
    else
        print_error "$test_type test failed with exit code $exit_code"
        return $exit_code
    fi
}

# Function to run mixed load test
run_mixed_test() {
    print_info "Running mixed endpoint load test..."
    
    local json_output="$RESULTS_DIR/mixed_${TIMESTAMP}.json"
    local summary_output="$RESULTS_DIR/mixed_${TIMESTAMP}_summary.txt"
    
    k6 run \
        --env TEST_HOST="$HOST" \
        --env JWT_TOKEN="$JWT_TOKEN" \
        --env TEST_TYPE="load" \
        --out json="$json_output" \
        --summary-export="$summary_output" \
        scenarios/mixed-load.js 2>&1 | tee "$RESULTS_DIR/mixed_${TIMESTAMP}_console.log"
    
    if [ $? -eq 0 ]; then
        print_success "Mixed endpoint test completed"
    else
        print_error "Mixed endpoint test failed"
        return 1
    fi
}

# Function to run endpoint-specific tests
run_endpoint_test() {
    print_info "Running endpoint-specific load test..."
    
    local json_output="$RESULTS_DIR/endpoint_${TIMESTAMP}.json"
    local summary_output="$RESULTS_DIR/endpoint_${TIMESTAMP}_summary.txt"
    
    k6 run \
        --env TEST_HOST="$HOST" \
        --env JWT_TOKEN="$JWT_TOKEN" \
        --out json="$json_output" \
        --summary-export="$summary_output" \
        scenarios/endpoint-specific.js 2>&1 | tee "$RESULTS_DIR/endpoint_${TIMESTAMP}_console.log"
    
    if [ $? -eq 0 ]; then
        print_success "Endpoint-specific test completed"
    else
        print_error "Endpoint-specific test failed"
        return 1
    fi
}

# Function to generate test report
generate_report() {
    print_info "Generating consolidated test report..."
    
    local report_file="$RESULTS_DIR/test_report_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# Load Test Report - $(date)

## Test Configuration
- **Host**: $HOST
- **Timestamp**: $TIMESTAMP
- **K6 Version**: $(k6 version --short)

## Test Results

$(ls -la $RESULTS_DIR/*${TIMESTAMP}* | while read -r line; do
    filename=$(echo $line | awk '{print $NF}')
    basename=$(basename "$filename")
    echo "- [$basename](./$basename)"
done)

## Key Metrics Summary

### Response Times
$(if [ -f "$RESULTS_DIR/load_${TIMESTAMP}_summary.txt" ]; then
    echo "#### Load Test"
    grep -E "(http_req_duration|http_req_failed)" "$RESULTS_DIR/load_${TIMESTAMP}_summary.txt" || echo "No summary available"
fi)

### Recommendations
$(if [ -f "$RESULTS_DIR/load_${TIMESTAMP}_console.log" ]; then
    grep -A 5 "🔧\|✅" "$RESULTS_DIR/load_${TIMESTAMP}_console.log" | head -10 || echo "Check individual test logs for recommendations"
fi)

## Files Generated
$(ls -1 $RESULTS_DIR/*${TIMESTAMP}* | sort)

---
Generated by K6 Load Testing Suite for InsightCollector
EOF
    
    print_success "Test report generated: $report_file"
}

# Main execution
main() {
    # Parse command line arguments
    TEST_TYPE=${1:-"load"}
    HOST=${2:-$DEFAULT_HOST}
    JWT_TOKEN=${3:-$DEFAULT_JWT}
    
    # Show usage if requested
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        show_usage
    fi
    
    # Show configuration
    print_info "Load Test Configuration:"
    echo "  • Test Type: $TEST_TYPE"
    echo "  • Target Host: $HOST"
    echo "  • JWT Token: ${JWT_TOKEN:0:20}... (${#JWT_TOKEN} chars)"
    echo ""
    
    # Check prerequisites
    check_prerequisites
    
    # Health check
    health_check
    
    # Run tests based on type
    case $TEST_TYPE in
        "smoke")
            run_test "smoke" "scenarios/mixed-load.js"
            ;;
        "load")
            run_test "load" "scenarios/mixed-load.js"
            ;;
        "stress")
            run_test "stress" "scenarios/mixed-load.js"
            ;;
        "spike")
            run_test "spike" "scenarios/mixed-load.js"
            ;;
        "mixed")
            run_mixed_test
            ;;
        "endpoint")
            run_endpoint_test
            ;;
        "all")
            print_info "Running all test types sequentially..."
            
            run_test "smoke" "scenarios/mixed-load.js" && \
            run_test "load" "scenarios/mixed-load.js" && \
            run_test "stress" "scenarios/mixed-load.js" && \
            run_mixed_test && \
            run_endpoint_test
            
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
    
    # Generate consolidated report
    generate_report
    
    print_success "Load testing completed! 🎉"
    print_info "Check the results in the '$RESULTS_DIR' directory"
    
    # Show next steps
    echo ""
    print_info "Next steps:"
    echo "  1. Analyze the results in $RESULTS_DIR/"
    echo "  2. Check your InfluxDB and Redis for backend performance impact"
    echo "  3. Monitor your worker queues: ./app worker status"
    echo "  4. Adjust concurrency if needed: ./app worker concurrency [number]"
}

# Run main function with all arguments
main "$@"