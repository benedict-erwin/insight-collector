// Endpoint-Specific Load Test - Focus on Individual Endpoints
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { config } from '../config/test-config.js';
import { payloadGenerators } from '../data/payload-generators.js';

// Custom metrics per endpoint
const requestsPerEndpoint = new Counter('requests_per_endpoint');
const responseTimePerEndpoint = new Trend('response_time_per_endpoint');
const errorRatePerEndpoint = new Rate('error_rate_per_endpoint');

// Test configuration
export let options = {
  scenarios: {
    // User Activities - Heavy Load (most frequent endpoint)
    userActivitiesLoad: {
      executor: 'constant-arrival-rate',
      rate: 20, // 20 requests per second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 10,
      maxVUs: 30,
      tags: { endpoint: 'user-activities' },
      env: { TARGET_ENDPOINT: 'user-activities' },
    },
    
    // Transaction Events - Medium Load
    transactionEventsLoad: {
      executor: 'constant-arrival-rate',
      rate: 12, // 12 requests per second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 6,
      maxVUs: 20,
      tags: { endpoint: 'transaction-events' },
      env: { TARGET_ENDPOINT: 'transaction-events' },
      startTime: '30s', // Start 30s after user-activities
    },

    // Security Events - Burst Load (simulating security incidents)
    securityEventsBurst: {
      executor: 'ramping-arrival-rate',
      startRate: 5,
      timeUnit: '1s',
      preAllocatedVUs: 5,
      maxVUs: 25,
      stages: [
        { duration: '1m', target: 5 },   // Normal security monitoring
        { duration: '30s', target: 25 }, // Security incident spike
        { duration: '1m', target: 25 },  // Sustained incident handling  
        { duration: '30s', target: 5 },  // Return to normal
      ],
      tags: { endpoint: 'security-events' },
      env: { TARGET_ENDPOINT: 'security-events' },
      startTime: '1m', // Start 1m after user-activities
    },

    // Callback Logs - Steady Load (external webhook simulation)  
    callbackLogsSteady: {
      executor: 'constant-vus',
      vus: 8,
      duration: '4m',
      tags: { endpoint: 'callback-logs' },
      env: { TARGET_ENDPOINT: 'callback-logs' },
      startTime: '1m30s', // Start 1m30s after user-activities
    },
  },
  
  thresholds: {
    // Overall thresholds
    'http_req_duration': ['p(95)<2000'],
    'http_req_failed': ['rate<0.05'],
    
    // Per-endpoint thresholds
    'http_req_duration{endpoint:user-activities}': ['p(95)<800'],
    'http_req_duration{endpoint:transaction-events}': ['p(95)<1000'],
    'http_req_duration{endpoint:security-events}': ['p(95)<1200'],
    'http_req_duration{endpoint:callback-logs}': ['p(95)<1500'],
    
    'http_req_failed{endpoint:user-activities}': ['rate<0.02'],
    'http_req_failed{endpoint:transaction-events}': ['rate<0.03'],
    'http_req_failed{endpoint:security-events}': ['rate<0.05'],
    'http_req_failed{endpoint:callback-logs}': ['rate<0.05'],
  },
};

// Authentication headers
function getAuthHeaders(endpoint) {
  return {
    'Authorization': `Bearer ${config.auth.jwt_token}`,
    'Content-Type': 'application/json',
    'X-Test-Run': `k6-endpoint-${endpoint}-${Date.now()}`,
    'X-Load-Test': 'endpoint-specific',
  };
}

// Main test function
export default function() {
  const targetEndpoint = __ENV.TARGET_ENDPOINT;
  
  if (!targetEndpoint) {
    throw new Error('TARGET_ENDPOINT environment variable not set');
  }

  // Generate payload for the specific endpoint
  const generator = payloadGenerators[targetEndpoint];
  if (!generator) {
    throw new Error(`No payload generator found for endpoint: ${targetEndpoint}`);
  }

  const payload = generator();
  
  // Add endpoint-specific test metadata
  payload._test_metadata = {
    k6_vu: __VU,
    k6_iteration: __ITER,
    test_type: 'endpoint-specific',
    target_endpoint: targetEndpoint,
    scenario_name: __ENV.K6_SCENARIO_NAME || 'unknown',
  };

  const url = `${config.base.host}/v1/${targetEndpoint}/insert`;
  const headers = getAuthHeaders(targetEndpoint);
  
  const startTime = Date.now();
  
  const response = http.post(url, JSON.stringify(payload), {
    headers: headers,
    timeout: config.base.timeout,
    tags: { 
      endpoint: targetEndpoint,
      scenario: __ENV.K6_SCENARIO_NAME || 'unknown',
    },
  });
  
  const endTime = Date.now();
  const duration = endTime - startTime;

  // Record metrics
  requestsPerEndpoint.add(1, { endpoint: targetEndpoint });
  responseTimePerEndpoint.add(duration, { endpoint: targetEndpoint });

  // Endpoint-specific checks
  const checks = getEndpointSpecificChecks(targetEndpoint);
  const checkResult = check(response, checks);

  // Record error rate
  errorRatePerEndpoint.add(!checkResult, { endpoint: targetEndpoint });

  // Enhanced logging for failures
  if (!checkResult || response.status >= 400) {
    console.error(`❌ ${targetEndpoint} failed:`, {
      status: response.status,
      duration: duration,
      body: response.body.substring(0, 300),
      payload_size: JSON.stringify(payload).length,
      iteration: __ITER,
    });
  } else if (__ITER % 100 === 0) {
    // Log success every 100 iterations
    console.log(`✅ ${targetEndpoint} - ${__ITER} requests completed (${duration}ms)`);
  }

  // Endpoint-specific sleep patterns
  sleep(getEndpointSleepDuration(targetEndpoint));
}

// Get endpoint-specific validation checks
function getEndpointSpecificChecks(endpoint) {
  const baseChecks = {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'has request_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.request_id && body.request_id.length > 0;
      } catch {
        return false;
      }
    },
    'response time acceptable': (r) => r.timings.duration < 3000,
  };

  const endpointSpecificChecks = {
    'user-activities': {
      ...baseChecks,
      'user activities response valid': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success && body.data && body.data.job_id;
        } catch {
          return false;
        }
      },
      'fast response': (r) => r.timings.duration < 800, // User activities should be fastest
    },
    
    'transaction-events': {
      ...baseChecks,
      'transaction response valid': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success && body.data && body.data.job_id;
        } catch {
          return false;
        }
      },
      'financial data secure': (r) => r.headers['X-Content-Type-Options'] !== undefined,
    },
    
    'security-events': {
      ...baseChecks,
      'security response valid': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success && body.data;
        } catch {
          return false;
        }
      },
      'security headers present': (r) => {
        // Check for security-related headers
        return r.headers['X-Request-ID'] !== undefined;
      },
    },
    
    'callback-logs': {
      ...baseChecks,
      'callback response valid': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success && body.data;
        } catch {
          return false;
        }
      },
      'webhook processing': (r) => r.timings.duration < 1500, // Callbacks can be slower
    },
  };

  return endpointSpecificChecks[endpoint] || baseChecks;
}

// Get endpoint-specific sleep duration
function getEndpointSleepDuration(endpoint) {
  const sleepDurations = {
    'user-activities': Math.random() * 0.5 + 0.2,    // 0.2-0.7s (frequent user actions)
    'transaction-events': Math.random() * 2 + 1,     // 1-3s (less frequent transactions)  
    'security-events': Math.random() * 1 + 0.5,      // 0.5-1.5s (security monitoring)
    'callback-logs': Math.random() * 3 + 2,          // 2-5s (external webhook timing)
  };

  return sleepDurations[endpoint] || 1;
}

// Setup function
export function setup() {
  console.log(`🎯 Starting Endpoint-Specific Load Test`);
  console.log(`🏥 Health check...`);
  
  const healthCheck = http.get(`${config.base.host}/v1/health/live`);
  
  if (healthCheck.status !== 200) {
    throw new Error(`❌ Health check failed: ${healthCheck.status}`);
  }
  
  console.log(`✅ Service ready for endpoint-specific testing`);
  console.log(`📊 Test scenarios:`);
  console.log(`   • User Activities: 20 req/s for 5m`);
  console.log(`   • Transaction Events: 12 req/s for 5m`); 
  console.log(`   • Security Events: 5-25 req/s burst pattern`);
  console.log(`   • Callback Logs: 8 constant VUs for 4m`);
  
  return { startTime: Date.now() };
}

// Teardown function
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  
  console.log(`\n🏁 Endpoint-Specific Test Completed`);
  console.log(`⏱️  Duration: ${duration.toFixed(2)} seconds`);
  console.log(`📈 Check individual endpoint performance above`);
}