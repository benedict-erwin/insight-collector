// Mixed Load Test - All 4 Endpoints with Realistic Traffic Distribution
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { config, getConfig } from '../config/test-config.js';
import { payloadGenerators } from '../data/payload-generators.js';

// Custom metrics for detailed monitoring
const jobEnqueueDuration = new Trend('job_enqueue_duration');
const endpointCounter = new Counter('requests_per_endpoint');
const errorRate = new Rate('error_rate');

// Test configuration based on environment
export let options = getConfig(__ENV.TEST_TYPE || 'load');

// Weighted endpoint distribution (realistic traffic patterns)
const endpointWeights = {
  'user-activities': 0.40,    // 40% - Most frequent (every user action)
  'transaction-events': 0.25,  // 25% - Moderate (financial operations)
  'security-events': 0.20,    // 20% - Security monitoring
  'callback-logs': 0.15,      // 15% - External webhooks/callbacks
};

// Create weighted endpoint selector
function selectRandomEndpoint() {
  const rand = Math.random();
  let cumulativeWeight = 0;
  
  for (const [endpoint, weight] of Object.entries(endpointWeights)) {
    cumulativeWeight += weight;
    if (rand <= cumulativeWeight) {
      return endpoint;
    }
  }
  return 'user-activities'; // fallback
}

// Authentication headers
function getAuthHeaders() {
  return {
    'Authorization': `Bearer ${config.auth.jwt_token}`,
    'Content-Type': 'application/json',
    'X-Test-Run': `k6-${__ENV.TEST_TYPE || 'load'}-${Date.now()}`,
  };
}

// Main test function
export default function() {
  const endpoint = selectRandomEndpoint();
  const payload = payloadGenerators[endpoint]();
  
  // Add test metadata to payload for tracking
  payload._test_metadata = {
    k6_vu: __VU,
    k6_iteration: __ITER,
    test_type: __ENV.TEST_TYPE || 'load',
    endpoint_type: endpoint,
  };

  const url = `${config.base.host}/v1/${endpoint}/insert`;
  const headers = getAuthHeaders();
  
  // Measure job enqueue time (simulate measuring time to enqueue)
  const enqueueStart = Date.now();
  
  const response = http.post(url, JSON.stringify(payload), {
    headers: headers,
    timeout: config.base.timeout,
    tags: { 
      endpoint: endpoint,
      test_type: __ENV.TEST_TYPE || 'load'
    },
  });
  
  const enqueueEnd = Date.now();
  jobEnqueueDuration.add(enqueueEnd - enqueueStart);

  // Increment endpoint counter
  endpointCounter.add(1, { endpoint: endpoint });

  // Check response
  const checkResult = check(response, {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'response has success field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hasOwnProperty('success');
      } catch {
        return false;
      }
    },
    'response time < 2000ms': (r) => r.timings.duration < 2000,
    'job_id returned': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success && body.data && body.data.job_id;
      } catch {
        return false;
      }
    },
  });

  // Track error rate
  errorRate.add(!checkResult);

  // Log details for failed requests
  if (!checkResult || response.status >= 400) {
    console.error(`❌ ${endpoint} failed:`, {
      status: response.status,
      url: url,
      body: response.body ? response.body.substring(0, 200) : 'No response body',
      duration: response.timings.duration,
    });
  }


  // Variable sleep between requests (0.5 - 2 seconds)
  sleep(Math.random() * 1.5 + 0.5);
}

// Setup function - runs once before test starts
export function setup() {
  console.log(`🚀 Starting K6 Load Test`);
  console.log(`📊 Test Type: ${__ENV.TEST_TYPE || 'load'}`);
  console.log(`🎯 Target Host: ${config.base.host}`);
  console.log(`⚖️  Endpoint Distribution:`, endpointWeights);
  
  // Health check before starting load test
  const healthCheck = http.get(`${config.base.host}/v1/health/live`);
  
  if (healthCheck.status !== 200) {
    throw new Error(`❌ Health check failed: ${healthCheck.status} ${healthCheck.body}`);
  }
  
  console.log(`✅ Health check passed - Service is ready for load testing`);
  
  return {
    startTime: Date.now(),
    testConfig: config,
  };
}

// Teardown function - runs once after test completes  
export function teardown(data) {
  const endTime = Date.now();
  const duration = (endTime - data.startTime) / 1000;
  
  console.log(`\n🏁 Load Test Completed`);
  console.log(`⏱️  Total Duration: ${duration.toFixed(2)} seconds`);
  console.log(`📈 Check the results above for performance metrics`);
  console.log(`📊 Monitor your InfluxDB and Redis for backend performance`);
  
  // Optional: Send test completion webhook (if configured)
  if (__ENV.WEBHOOK_URL) {
    const summary = {
      test_type: __ENV.TEST_TYPE || 'load',
      duration_seconds: duration,
      completed_at: new Date().toISOString(),
      target_host: config.base.host,
    };
    
    http.post(__ENV.WEBHOOK_URL, JSON.stringify(summary), {
      headers: { 'Content-Type': 'application/json' },
    });
  }
}