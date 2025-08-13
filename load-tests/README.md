# K6 Load Testing Suite for InsightCollector

Comprehensive **Docker-based** load testing framework for testing the InsightCollector service performance with realistic traffic patterns and dynamic payloads. **No local K6 installation required!**

## 🎯 Testing Modes Overview

The test suite provides **flexible testing approaches** for different analysis needs:

### 1. **Mixed Endpoint Testing** (All endpoints together)
- **Purpose**: Real-world usage simulation  
- **Traffic Distribution**: 40% user-activities, 25% transaction-events, 20% security-events, 15% callback-logs
- **Use Case**: Overall system performance validation

### 2. **Individual Endpoint Testing** (One endpoint at a time)
- **Purpose**: Deep endpoint-specific analysis
- **Load Patterns**: Customized per endpoint type
- **Use Case**: Optimization targeting and SLA validation

### 3. **Endpoint Comparison Testing** (All endpoints individually + comparison)
- **Purpose**: Performance ranking and optimization prioritization
- **Output**: Detailed comparison report with recommendations
- **Use Case**: Capacity planning and resource allocation

## 🚀 Quick Start (Docker-based)

### Prerequisites
**Only Docker is required** - no K6 installation needed:

```bash
# Check Docker is available
docker --version
docker compose version  # or docker-compose --version
```

### Basic Testing Commands

```bash
cd load-tests

# 🔥 RECOMMENDED: Start with smoke test
./docker-run.sh smoke

# Mixed endpoint testing (all endpoints together)
./docker-run.sh mixed

# Individual endpoint testing with comparison  
./single-endpoint-test.sh compare

# Specific endpoint deep dive
./single-endpoint-test.sh user-activities

# Full comprehensive testing
./docker-run.sh all
```

## 📊 Testing Scenarios Explained

### Mixed Load Testing (`./docker-run.sh [scenario]`)

| Scenario | Duration | Load Pattern | Purpose |
|----------|----------|--------------|---------|
| **smoke** | 30 seconds | 1 user | ✅ Basic functionality validation |
| **load** | 9 minutes | Up to 10 users | 📈 Normal operating conditions |
| **stress** | 16 minutes | Up to 50 users | 🔥 Peak traffic simulation |
| **spike** | 1 minute | 1→250→1 users | ⚡ Traffic burst recovery |
| **mixed** | Custom | Realistic distribution | 🌐 Production simulation |

### Individual Endpoint Testing (`./single-endpoint-test.sh [endpoint]`)

| Endpoint | Test Pattern | Duration | Focus Area |
|----------|-------------|----------|------------|
| **user-activities** | Intensive ramping | 8 minutes | High-frequency user actions |
| **transaction-events** | Steady arrival rate | 5 minutes | Financial operation consistency |
| **security-events** | Burst pattern | 5 minutes | Security incident simulation |
| **callback-logs** | Constant load | 4 minutes | External webhook processing |

## 🛠 Detailed Usage Guide

### 1. Mixed Endpoint Testing

```bash
# Quick validation (30 seconds)
./docker-run.sh smoke

# Production readiness (9 minutes)
./docker-run.sh load

# Peak capacity testing (16 minutes)  
./docker-run.sh stress

# With real-time metrics and visualization
./docker-run.sh load --with-metrics
# Access: Grafana (http://localhost:3001) InfluxDB (http://localhost:8087)

# Traffic spike simulation
./docker-run.sh spike

# Realistic production traffic pattern
./docker-run.sh mixed
```

### 2. Individual Endpoint Analysis

```bash
# Test single endpoint with intensive load
./single-endpoint-test.sh user-activities
./single-endpoint-test.sh transaction-events  
./single-endpoint-test.sh security-events
./single-endpoint-test.sh callback-logs

# Test all endpoints individually + generate comparison
./single-endpoint-test.sh compare

# Test all endpoints sequentially (no comparison)
./single-endpoint-test.sh all

# With metrics collection and analysis
./single-endpoint-test.sh compare --with-metrics --analyze
```

### 3. Custom Host Testing

```bash
# Local service (Docker networking)
./docker-run.sh load http://host.docker.internal:8080

# Local service (direct)
./docker-run.sh load http://localhost:8080

# Remote staging
./docker-run.sh smoke https://staging.insight.com

# Individual endpoint on custom host
./single-endpoint-test.sh user-activities https://staging.insight.com
```

## 📈 Performance Thresholds & Validation

### Response Time Targets
- **Overall System**: 95th percentile < 2000ms
- **User Activities**: 95th percentile < 800ms (most critical)
- **Transaction Events**: 95th percentile < 1000ms
- **Security Events**: 95th percentile < 1200ms  
- **Callback Logs**: 95th percentile < 1500ms

### Error Rate Targets
- **Overall**: < 5% error rate
- **User Activities**: < 2% (critical user path)
- **Other Endpoints**: < 5%

### What K6 Actually Measures
**✅ HTTP Layer Performance (Real Metrics):**
- **HTTP Response Times**: Request-response cycle from K6 perspective
- **Job Enqueue Duration**: Time to dispatch jobs to Redis queue (HTTP → Queue only)
- **Per-Endpoint Performance**: Individual endpoint response metrics
- **Traffic Distribution**: Validation of realistic load patterns
- **Error Rates**: Request success/failure tracking

**❌ Background Processing (Not Measured):**
- Background worker job processing times
- Actual InfluxDB write performance
- Queue processing latency
- Database transaction times

**📝 Important Note**: K6 tests the HTTP API performance, not the background job processing. A successful HTTP response means the job was enqueued, but doesn't measure how long the background worker takes to process it.

## 📁 Results Analysis

### Generated Files Structure
```
results/
├── mixed_20241201_143022.json                     # Mixed test raw data
├── mixed_20241201_143022_summary.txt              # Mixed test summary
├── single_user-activities_20241201_144530.json    # Individual endpoint data
├── single_transaction-events_20241201_145001.json
├── endpoint_comparison_20241201_145500.md         # Comparison report
└── *_console.log                                  # Detailed execution logs
```

### Key Analysis Commands
```bash
# View latest test summary
ls -la results/ | tail -5

# Check mixed endpoint results
cat results/mixed_*_summary.txt | grep -E "(http_req_duration|http_req_failed)"

# View endpoint comparison
cat results/endpoint_comparison_*.md

# Analyze specific endpoint performance
grep "user_activities" results/single_*_summary.txt
```

## 🔧 Advanced Configuration

### Environment Variables
```bash
# Custom configuration
export TEST_HOST="https://your-service.com"  
export TEST_TYPE="stress"

# Run with custom settings
./docker-run.sh stress
```

### Metrics and Monitoring Setup
```bash
# Start test with full monitoring stack
./docker-run.sh load --with-metrics

# Access monitoring dashboards:
# • Grafana: http://localhost:3001 (admin/k6admin)
# • InfluxDB: http://localhost:8087 (k6/k6password)

# Individual endpoint with metrics
./single-endpoint-test.sh compare --with-metrics
```

### Cleanup and Management
```bash
# View running containers
docker ps

# Show test logs
./docker-run.sh --logs

# Complete cleanup (containers, volumes, networks)
./docker-run.sh --cleanup
```

## 🎯 Recommended Testing Workflows

### 1. **Initial Development Testing**
```bash
# Quick functionality validation
./docker-run.sh smoke

# If smoke passes, check normal load
./docker-run.sh load
```

### 2. **Pre-Production Validation**
```bash
# Full endpoint analysis
./single-endpoint-test.sh compare --analyze

# System-wide stress testing
./docker-run.sh stress --with-metrics

# Traffic spike resilience
./docker-run.sh spike
```

### 3. **Performance Optimization Workflow**
```bash
# 1. Identify bottlenecks
./single-endpoint-test.sh compare

# 2. Focus on problematic endpoints
./single-endpoint-test.sh security-events --with-metrics

# 3. Validate system-wide improvements
./docker-run.sh mixed

# 4. Confirm under stress
./docker-run.sh stress
```

### 4. **Production Readiness Checklist**
```bash
# ✅ All scenarios pass
./docker-run.sh all

# ✅ Individual endpoints meet SLA
./single-endpoint-test.sh compare

# ✅ Traffic spikes handled gracefully  
./docker-run.sh spike

# ✅ Sustained load performance
./docker-run.sh stress --with-metrics
```

## 🚨 Performance Interpretation Guide

### ✅ **Good Performance Indicators**
- Response times consistently under thresholds
- Error rates < 2% for critical endpoints
- Stable performance during sustained load
- Quick recovery from traffic spikes
- Job enqueue times remain consistent

### ⚠️ **Warning Signs**
- Response times trending upward during test
- Error rates 2-5% range
- Gradual performance degradation
- Job enqueue times increasing
- Memory usage increasing linearly

### ❌ **Critical Issues**  
- Response times > 2 seconds consistently
- Error rates > 5%
- Service becoming unresponsive
- Job enqueue failures or timeouts
- Memory leaks or resource exhaustion

## 🔧 Optimization Actions Based on Results

### High Response Times
```bash
# Check worker configuration
./app worker list

# Increase concurrency
./app worker concurrency 25

# Monitor queue distribution
./app worker status
```

### High Error Rates
```bash
# Check application health
curl http://localhost:8080/v1/health

# Review service logs
docker logs insight-collector -f

# Verify dependencies
curl http://localhost:8086/health  # InfluxDB
```

### High Job Enqueue Times
```bash
# Check application load and Redis performance
curl http://localhost:8080/v1/health

# Monitor worker status (if available)
./app worker status

# Scale workers if background processing is the bottleneck  
./app worker concurrency 30
```

## 🌐 Docker Networking Notes

**Important for local testing:**

- ✅ **Use**: `http://host.docker.internal:8080` (Docker can reach your host services)
- ❌ **Avoid**: `http://localhost:8080` (refers to container, not host)

**For remote testing:**
- ✅ **Use**: Full URLs like `https://staging.insight.com`

## 🛡️ Non-Auth Testing Configuration

**Current setup is optimized for testing without authentication:**

- JWT token automatically set to: `dummy-token-for-non-auth-testing`
- All requests include this dummy token in Authorization header
- Your endpoints should ignore/bypass auth validation during testing
- When implementing auth: update `JWT_TOKEN` environment variable

## 📚 Troubleshooting Guide

### Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| **"Health check failed"** | Verify service is running: `curl http://localhost:8080/v1/health/live` |
| **"Docker compose not found"** | Install Docker Compose or use `docker-compose` instead |
| **"Permission denied"** | Run: `chmod +x *.sh` |
| **"No route to host"** | Use `host.docker.internal:8080` for local services |
| **High memory usage** | Reduce test concurrency or add Docker resource limits |

### Debug Commands
```bash
# Check container status
docker ps -a

# View detailed logs
./docker-run.sh --logs

# Test network connectivity
docker run --rm alpine ping host.docker.internal

# Verify service health
curl http://localhost:8080/v1/health/live
```

## 🎯 Quick Reference Commands

```bash
# Essential testing commands
./docker-run.sh smoke                    # Quick validation
./docker-run.sh load                     # Normal load test  
./single-endpoint-test.sh compare        # Endpoint analysis
./docker-run.sh stress --with-metrics    # Peak load + monitoring
./docker-run.sh --cleanup               # Full cleanup

# Monitoring and analysis
./single-endpoint-test.sh user-activities --with-metrics
open http://localhost:3001               # Grafana dashboard
cat results/endpoint_comparison_*.md     # Performance comparison
```

---

## 🚀 Ready to Start Testing!

**Step 1**: Ensure your InsightCollector service is running
```bash
curl http://localhost:8080/v1/health/live
```

**Step 2**: Run your first test  
```bash
cd load-tests
./docker-run.sh smoke
```

**Step 3**: Analyze results and scale up testing based on your needs!

For questions or issues, check the `results/` directory for detailed logs and performance data.

**Happy Load Testing! 🎯**