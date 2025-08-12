# 🐳 Docker-based K6 Load Testing

No need to install K6 locally! Everything runs in Docker containers.

## 🚀 Quick Start (Non-Auth Testing)

```bash
cd load-tests

# Basic smoke test (30 seconds, 1 user)
./docker-run.sh smoke

# Load test (9 minutes, up to 10 users)  
./docker-run.sh load

# Stress test (16 minutes, up to 50 users)
./docker-run.sh stress

# With metrics collection (InfluxDB + Grafana)
./docker-run.sh load --with-metrics

# With post-test analysis
./docker-run.sh load --analyze

# Run all tests
./docker-run.sh all
```

## 📁 What's Included

### Core Services
- **K6 Runner**: Executes load tests in container
- **Results Storage**: Persistent results in `./results/` directory
- **Health Checking**: Automatic service availability verification

### Optional Services (with --with-metrics)
- **InfluxDB v1.8**: Stores detailed K6 HTTP metrics
- **Grafana**: Visualizes test results in real-time with auto-provisioned dashboards
- **Analyzer**: Post-test analysis and recommendations

## 🛠 Prerequisites

Only Docker is required:

```bash
# Check Docker is installed and running
docker --version
docker info

# For Linux, might need Docker Compose plugin
docker compose version
```

## 📊 Docker vs Local Installation

| Feature | Docker | Local K6 |
|---------|--------|----------|
| **Installation** | Just Docker | Install K6 + dependencies |
| **Consistency** | Same environment everywhere | Depends on local setup |  
| **Isolation** | Isolated from host | Uses host resources directly |
| **Networking** | `host.docker.internal:8080` | `localhost:8080` |
| **Results** | Persistent volume | Local filesystem |
| **Metrics** | Optional InfluxDB + Grafana | Manual setup needed |

## 🌐 Network Configuration

### Testing Local Services

For testing services running on your host machine:

```bash
# ✅ CORRECT - Docker can reach host services
./docker-run.sh load http://host.docker.internal:8080

# ❌ WRONG - localhost refers to container, not host
./docker-run.sh load http://localhost:8080
```

### Testing Remote Services

For testing remote services:

```bash
# Remote staging
./docker-run.sh load https://staging.insight.com

# Remote production (be careful!)
./docker-run.sh smoke https://api.insight.com
```

## 🔧 Common Usage Patterns

### Basic Testing Workflow

```bash
# 1. Start with smoke test
./docker-run.sh smoke

# 2. Run load test if smoke passes
./docker-run.sh load

# 3. Stress test for peak capacity
./docker-run.sh stress

# 4. Check results
ls -la results/
```

### Development Testing

```bash
# Quick validation after code changes
./docker-run.sh smoke http://host.docker.internal:8080

# Monitor during test
watch -n 2 'docker stats --no-stream'
```

### Production Readiness Testing

```bash
# Full test suite with metrics
./docker-run.sh all --with-metrics --analyze

# Check metrics
open http://localhost:3001  # Grafana dashboard
```

## 📈 Metrics & Monitoring

### Start Metrics Collection

```bash
./docker-run.sh load --with-metrics
```

This starts:
- **InfluxDB v1.8** at `http://localhost:8087` (k6/k6password) 
- **Grafana** at `http://localhost:3001` (admin/k6admin) with K6 dashboard pre-loaded

### Access Dashboards

```bash
# Grafana dashboards
open http://localhost:3001

# InfluxDB metrics
open http://localhost:8087
```

### Stop Metrics Services

```bash
docker compose --profile metrics down
```

## 📊 Results Analysis

### View Results

```bash
# List all results
ls -la results/

# View summary of latest test
cat results/*_summary.txt | tail -20

# Analyze JSON results  
./docker-run.sh --analyze
```

### Results Structure

```
results/
├── load_20241201_143022.json          # Raw K6 metrics
├── load_20241201_143022_summary.txt   # Human-readable summary
├── stress_20241201_144530.json        # Stress test results
└── analysis_report.txt                # Automated analysis
```

## 🐛 Troubleshooting

### Container Issues

```bash
# Check Docker status
docker ps

# View K6 logs
./docker-run.sh --logs

# Clean up everything
./docker-run.sh --cleanup
```

### Network Issues

```bash
# Test host connectivity from container
docker run --rm alpine ping host.docker.internal

# Check if your service is accessible
curl http://localhost:8080/v1/health/live
```

### Permission Issues

```bash
# Fix script permissions
chmod +x docker-run.sh

# Fix results directory permissions (if needed)
sudo chown -R $USER:$USER results/
```

## 🔄 Advanced Usage

### Custom Test Configuration

```bash
# Set environment variables
export TEST_HOST="https://your-service.com"
export TEST_TYPE="stress"

# Run with custom settings
./docker-run.sh stress
```

### Multiple Concurrent Tests

```bash
# Run different test types in parallel (careful with resources!)
./docker-run.sh smoke &
./docker-run.sh load --with-metrics &
wait
```

### Integration with CI/CD

```bash
#!/bin/bash
# ci-load-test.sh

set -e

# Start services
docker-compose up -d insight-collector

# Wait for service to be ready
sleep 30

# Run load test
cd load-tests
./docker-run.sh smoke

# Check exit code
if [ $? -eq 0 ]; then
    echo "✅ Load test passed"
else
    echo "❌ Load test failed"
    exit 1
fi

# Cleanup
./docker-run.sh --cleanup
```

## 💡 Tips & Best Practices

### Resource Management

```bash
# Monitor Docker resource usage during tests
docker stats --no-stream

# Limit container resources if needed
docker run --memory="512m" --cpus="1.0" grafana/k6:latest
```

### Test Isolation

```bash
# Clean state before important tests
./docker-run.sh --cleanup
docker system prune -f

# Run test
./docker-run.sh load --with-metrics
```

### Data Persistence

```bash
# Results are automatically persistent in ./results/
# But metrics data is lost unless you backup volumes

# Backup metrics
docker run --rm -v insight-k6-influxdb-data:/data -v $(pwd):/backup alpine tar czf /backup/influxdb-backup.tar.gz /data
```

## 🔗 Integration Examples

### With Your Application

```bash
# In your development workflow
make start-services          # Start your app
cd load-tests
./docker-run.sh smoke         # Quick validation
./docker-run.sh load          # Full load test
make stop-services           # Clean shutdown
```

### With Monitoring

```bash
# Start your monitoring stack
docker-compose up -d grafana influxdb

# Run tests with metrics
cd load-tests  
./docker-run.sh load --with-metrics

# View combined metrics in your Grafana
open http://localhost:3000  # Your app metrics  
open http://localhost:3001  # K6 test metrics
```

## 🎯 Non-Auth Testing Notes

**Important**: Current configuration is set for non-auth testing:

- JWT token is set to `dummy-token-for-non-auth-testing`
- All requests include this dummy token in Authorization header
- Your endpoints should ignore/skip auth validation for testing

When you implement authentication:
1. Generate real JWT tokens
2. Update `JWT_TOKEN` environment variable  
3. Enable auth middleware in your routes
4. Re-run tests to validate auth performance impact

---

**Ready to test? 🚀**

```bash
cd load-tests
./docker-run.sh smoke  # Start here!
```