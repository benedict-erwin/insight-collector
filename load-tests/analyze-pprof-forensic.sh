#!/bin/bash

# pprof Forensic Analysis Script
# Automatically analyzes collected pprof profiles to identify bottlenecks

set -e

# Configuration
if [[ -z "$1" ]]; then
    echo "Usage: $0 <pprof_directory>"
    echo "Example: $0 results/pprof_20250813_150000"
    exit 1
fi

PPROF_DIR="$1"
ANALYSIS_DIR="$1/analysis"
REPORT_FILE="$ANALYSIS_DIR/forensic_report.md"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

print_info() { echo -e "${BLUE}📊 $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

if [[ ! -d "$PPROF_DIR" ]]; then
    print_error "pprof directory not found: $PPROF_DIR"
    exit 1
fi

mkdir -p "$ANALYSIS_DIR"

print_info "Starting pprof forensic analysis..."
print_info "Source: $PPROF_DIR"
print_info "Output: $ANALYSIS_DIR"

# Initialize report
cat > "$REPORT_FILE" << 'EOF'
# Asynq Bottleneck Forensic Analysis Report

Generated at: $(date)
Source directory: $(basename $PPROF_DIR)

## Executive Summary

This report analyzes pprof profiles collected during load testing to identify the root cause of Asynq performance bottlenecks.

EOF

echo "Source directory: $(basename $PPROF_DIR)" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Function to analyze CPU profiles
analyze_cpu_profiles() {
    print_info "Analyzing CPU profiles..."
    
    local cpu_dir="$PPROF_DIR/cpu"
    if [[ ! -d "$cpu_dir" || $(ls -1 "$cpu_dir"/*.prof 2>/dev/null | wc -l) -eq 0 ]]; then
        print_warning "No CPU profiles found"
        return
    fi
    
    echo "## CPU Analysis" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Get the latest CPU profile for detailed analysis
    local latest_cpu=$(ls -t "$cpu_dir"/*.prof | head -1)
    print_info "Analyzing latest CPU profile: $(basename $latest_cpu)"
    
    # Top functions consuming CPU
    echo "### Top CPU Consuming Functions" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text -nodecount=20 "$latest_cpu" 2>/dev/null | head -25 >> "$REPORT_FILE" || echo "CPU analysis failed" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Focus on Asynq-related functions
    echo "### Asynq-Specific CPU Usage" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text "$latest_cpu" 2>/dev/null | grep -i "asynq\|redis\|queue" | head -10 >> "$REPORT_FILE" || echo "No Asynq-specific CPU usage found" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Generate CPU flame graph (if available)
    if command -v go-torch &> /dev/null; then
        print_info "Generating CPU flame graph..."
        timeout 60 go-torch -f "$ANALYSIS_DIR/cpu_flamegraph.svg" "$latest_cpu" 2>/dev/null && \
            echo "### CPU Flame Graph: [cpu_flamegraph.svg](cpu_flamegraph.svg)" >> "$REPORT_FILE" || \
            echo "Flame graph generation failed" >> "$REPORT_FILE"
    fi
    
    print_success "CPU analysis completed"
}

# Function to analyze heap profiles
analyze_heap_profiles() {
    print_info "Analyzing heap profiles..."
    
    local heap_dir="$PPROF_DIR/heap"
    if [[ ! -d "$heap_dir" || $(ls -1 "$heap_dir"/*.prof 2>/dev/null | wc -l) -eq 0 ]]; then
        print_warning "No heap profiles found"
        return
    fi
    
    echo "## Memory Heap Analysis" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Get the latest heap profile
    local latest_heap=$(ls -t "$heap_dir"/*.prof | head -1)
    print_info "Analyzing latest heap profile: $(basename $latest_heap)"
    
    # Top memory consuming functions
    echo "### Top Memory Allocating Functions" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text -nodecount=20 "$latest_heap" 2>/dev/null | head -25 >> "$REPORT_FILE" || echo "Heap analysis failed" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Asynq memory usage
    echo "### Asynq Memory Usage" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text "$latest_heap" 2>/dev/null | grep -i "asynq\|redis\|queue" | head -10 >> "$REPORT_FILE" || echo "No Asynq-specific memory usage found" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    print_success "Heap analysis completed"
}

# Function to analyze goroutine profiles
analyze_goroutine_profiles() {
    print_info "Analyzing goroutine profiles..."
    
    local goroutine_dir="$PPROF_DIR/goroutine"
    if [[ ! -d "$goroutine_dir" || $(ls -1 "$goroutine_dir"/*.prof 2>/dev/null | wc -l) -eq 0 ]]; then
        print_warning "No goroutine profiles found"
        return
    fi
    
    echo "## Goroutine Analysis" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Get the latest goroutine profile
    local latest_goroutine=$(ls -t "$goroutine_dir"/*.prof | head -1)
    print_info "Analyzing latest goroutine profile: $(basename $latest_goroutine)"
    
    # Goroutine count and states
    echo "### Goroutine States" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text "$latest_goroutine" 2>/dev/null | head -20 >> "$REPORT_FILE" || echo "Goroutine analysis failed" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Look for blocking operations
    echo "### Potential Blocking Operations" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text "$latest_goroutine" 2>/dev/null | grep -E "chan|select|mutex|sync|wait|block|redis|asynq" | head -15 >> "$REPORT_FILE" || echo "No obvious blocking operations detected" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    print_success "Goroutine analysis completed"
}

# Function to analyze mutex profiles
analyze_mutex_profiles() {
    print_info "Analyzing mutex profiles..."
    
    local mutex_dir="$PPROF_DIR/mutex"
    if [[ ! -d "$mutex_dir" || $(ls -1 "$mutex_dir"/*.prof 2>/dev/null | wc -l) -eq 0 ]]; then
        print_warning "No mutex profiles found"
        return
    fi
    
    echo "## Mutex Contention Analysis" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Get the latest mutex profile
    local latest_mutex=$(ls -t "$mutex_dir"/*.prof | head -1)
    print_info "Analyzing latest mutex profile: $(basename $latest_mutex)"
    
    # Mutex contention hotspots
    echo "### Mutex Contention Hotspots" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text "$latest_mutex" 2>/dev/null | head -20 >> "$REPORT_FILE" || echo "No significant mutex contention detected" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    print_success "Mutex analysis completed"
}

# Function to analyze block profiles
analyze_block_profiles() {
    print_info "Analyzing block profiles..."
    
    local block_dir="$PPROF_DIR/block"
    if [[ ! -d "$block_dir" || $(ls -1 "$block_dir"/*.prof 2>/dev/null | wc -l) -eq 0 ]]; then
        print_warning "No block profiles found"
        return
    fi
    
    echo "## Blocking Operations Analysis" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Get the latest block profile
    local latest_block=$(ls -t "$block_dir"/*.prof | head -1)
    print_info "Analyzing latest block profile: $(basename $latest_block)"
    
    # Blocking operations
    echo "### Top Blocking Operations" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    timeout 30 go tool pprof -text "$latest_block" 2>/dev/null | head -20 >> "$REPORT_FILE" || echo "No significant blocking operations detected" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    print_success "Block analysis completed"
}

# Function to generate comparative analysis
generate_comparative_analysis() {
    print_info "Generating comparative analysis..."
    
    echo "## Comparative Analysis Over Time" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # CPU profile comparison
    local cpu_dir="$PPROF_DIR/cpu"
    if [[ -d "$cpu_dir" ]]; then
        local cpu_count=$(ls -1 "$cpu_dir"/*.prof 2>/dev/null | wc -l)
        echo "### CPU Profile Timeline ($cpu_count profiles collected)" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
        
        # Show profile collection times
        for profile in $(ls -t "$cpu_dir"/*.prof 2>/dev/null); do
            local filename=$(basename "$profile")
            local timestamp=${filename%.prof}
            local timestamp=${timestamp#cpu_}
            echo "$timestamp: $filename" >> "$REPORT_FILE"
        done | head -10
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
    fi
    
    # Memory growth analysis
    local heap_dir="$PPROF_DIR/heap"
    if [[ -d "$heap_dir" ]]; then
        local heap_count=$(ls -1 "$heap_dir"/*.prof 2>/dev/null | wc -l)
        echo "### Memory Profile Timeline ($heap_count profiles collected)" >> "$REPORT_FILE"
        echo '```' >> "$REPORT_FILE"
        
        # Show heap profile timeline
        for profile in $(ls -t "$heap_dir"/*.prof 2>/dev/null); do
            local filename=$(basename "$profile")
            local timestamp=${filename%.prof}
            local timestamp=${timestamp#heap_}
            echo "$timestamp: $filename" >> "$REPORT_FILE"
        done | head -10
        echo '```' >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
    fi
    
    print_success "Comparative analysis completed"
}

# Function to generate actionable recommendations
generate_recommendations() {
    print_info "Generating recommendations..."
    
    echo "## Recommendations" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    echo "### Immediate Actions" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "1. **Review CPU Analysis**: Focus on functions with highest CPU usage" >> "$REPORT_FILE"
    echo "2. **Check Memory Allocations**: Investigate high memory allocating functions" >> "$REPORT_FILE"
    echo "3. **Examine Blocking Operations**: Look for goroutines blocked on I/O or locks" >> "$REPORT_FILE"
    echo "4. **Analyze Mutex Contention**: Identify synchronization bottlenecks" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    echo "### Specific Areas to Investigate" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "- **Asynq job dispatching**: Check if job enqueueing is blocking" >> "$REPORT_FILE"
    echo "- **Redis connection pool**: Verify connection pool configuration and usage" >> "$REPORT_FILE"
    echo "- **JSON serialization**: Check if payload processing is expensive" >> "$REPORT_FILE"
    echo "- **HTTP handler execution**: Compare health vs Asynq endpoint performance" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    echo "### Performance Optimization Targets" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "- Reduce CPU usage in hot paths" >> "$REPORT_FILE"
    echo "- Minimize memory allocations" >> "$REPORT_FILE"
    echo "- Eliminate blocking operations in request handling" >> "$REPORT_FILE"
    echo "- Optimize Redis connection usage" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    print_success "Recommendations generated"
}

# Run all analyses
print_info "Starting comprehensive pprof analysis..."

analyze_cpu_profiles
analyze_heap_profiles
analyze_goroutine_profiles
analyze_mutex_profiles
analyze_block_profiles
generate_comparative_analysis
generate_recommendations

# Add footer to report
echo "---" >> "$REPORT_FILE"
echo "Report generated at: $(date)" >> "$REPORT_FILE"
echo "Analysis completed by: pprof forensic analyzer" >> "$REPORT_FILE"

print_success "Forensic analysis completed!"
print_info "Report saved to: $REPORT_FILE"
print_info "Analysis files in: $ANALYSIS_DIR"

echo ""
echo "📊 Quick Summary:"
echo "  • Report: $REPORT_FILE" 
echo "  • View with: cat '$REPORT_FILE' | less"
echo "  • Or open in browser/editor for better formatting"
echo ""

# Show quick preview of key findings
if [[ -f "$REPORT_FILE" ]]; then
    echo "🔍 Quick Preview - Top CPU Functions:"
    grep -A 10 "### Top CPU Consuming Functions" "$REPORT_FILE" 2>/dev/null | tail -10 || echo "  (Analysis details in full report)"
fi