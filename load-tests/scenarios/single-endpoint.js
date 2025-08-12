// Single Endpoint Load Test - Focus on One Specific Endpoint
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { config } from '../config/test-config.js';
import { payloadGenerators } from '../data/payload-generators.js';

// Get target endpoint from environment
const TARGET_ENDPOINT = __ENV.TARGET_ENDPOINT || 'user-activities';

// Validate endpoint
const validEndpoints = ['user-activities', 'transaction-events', 'security-events', 'callback-logs'];
if (!validEndpoints.includes(TARGET_ENDPOINT)) {
  throw new Error(`Invalid endpoint: ${TARGET_ENDPOINT}. Valid options: ${validEndpoints.join(', ')}`);
}

// Custom metrics for single endpoint
const endpointRequests = new Counter(`${TARGET_ENDPOINT.replace('-', '_')}_requests`);
const endpointErrors = new Rate(`${TARGET_ENDPOINT.replace('-', '_')}_error_rate`);
const endpointDuration = new Trend(`${TARGET_ENDPOINT.replace('-', '_')}_duration`);

// Endpoint-specific test configurations
const endpointConfigs = {
  'user-activities': {
    scenarios: {
      userActivitiesIntensive: {
        executor: 'ramping-vus',
        startVUs: 1,
        stages: [
          { duration: '30s', target: 5 },   // Warm up
          { duration: '2m', target: 15 },   // Ramp up to intensive load
          { duration: '3m', target: 15 },   // Sustain intensive load
          { duration: '1m', target: 25 },   // Peak load
          { duration: '2m', target: 25 },   // Sustain peak
          { duration: '30s', target: 0 },   // Ramp down
        ],
        tags: { endpoint: 'user-activities', test_mode: 'intensive' },
      },
    },
    thresholds: {
      'http_req_duration': ['p(95)<800'],
      'http_req_failed': ['rate<0.02'],
      'user_activities_duration': ['p(95)<600'],
      'user_activities_error_rate': ['rate<0.01'],
    },
    sleepDuration: () => Math.random() * 0.5 + 0.1, // 0.1-0.6s (frequent)
  },
  
  'transaction-events': {
    scenarios: {
      transactionEventsLoad: {
        executor: 'constant-arrival-rate',
        rate: 8, // 8 transactions per second
        timeUnit: '1s',
        duration: '5m',
        preAllocatedVUs: 5,
        maxVUs: 15,
        tags: { endpoint: 'transaction-events', test_mode: 'steady' },
      },
    },
    thresholds: {
      'http_req_duration': ['p(95)<1000'],
      'http_req_failed': ['rate<0.03'],
      'transaction_events_duration': ['p(95)<800'],
      'transaction_events_error_rate': ['rate<0.02'],
    },
    sleepDuration: () => Math.random() * 2 + 1, // 1-3s (moderate)
  },
  
  'security-events': {
    scenarios: {
      securityEventsBurst: {
        executor: 'ramping-arrival-rate',
        startRate: 2,
        timeUnit: '1s',
        preAllocatedVUs: 3,
        maxVUs: 20,
        stages: [
          { duration: '1m', target: 2 },    // Normal monitoring
          { duration: '30s', target: 15 },  // Security incident
          { duration: '2m', target: 15 },   // Incident handling
          { duration: '30s', target: 5 },   // Post-incident
          { duration: '1m', target: 5 },    // Return to normal
        ],
        tags: { endpoint: 'security-events', test_mode: 'burst' },
      },
    },
    thresholds: {
      'http_req_duration': ['p(95)<1200'],
      'http_req_failed': ['rate<0.05'],
      'security_events_duration': ['p(95)<1000'],
      'security_events_error_rate': ['rate<0.03'],
    },
    sleepDuration: () => Math.random() * 1.5 + 0.5, // 0.5-2s (variable)
  },
  
  'callback-logs': {
    scenarios: {
      callbackLogsWebhook: {
        executor: 'constant-vus',
        vus: 6,
        duration: '4m',
        tags: { endpoint: 'callback-logs', test_mode: 'webhook' },
      },
    },
    thresholds: {
      'http_req_duration': ['p(95)<1500'],
      'http_req_failed': ['rate<0.05'],
      'callback_logs_duration': ['p(95)<1200'],
      'callback_logs_error_rate': ['rate<0.04'],
    },
    sleepDuration: () => Math.random() * 3 + 2, // 2-5s (webhook timing)
  },
};

// Use endpoint-specific configuration
const endpointConfig = endpointConfigs[TARGET_ENDPOINT];
export let options = {
  scenarios: endpointConfig.scenarios,
  thresholds: endpointConfig.thresholds,
};

// Authentication headers
function getAuthHeaders() {
  return {
    'Authorization': `Bearer ${config.auth.jwt_token}`,
    'Content-Type': 'application/json',
    'X-Test-Run': `k6-single-${TARGET_ENDPOINT}-${Date.now()}`,
    'X-Load-Test': 'single-endpoint',
  };
}

// Main test function
export default function() {
  const generator = payloadGenerators[TARGET_ENDPOINT];
  const payload = generator();
  
  // Add single-endpoint test metadata
  payload._test_metadata = {
    k6_vu: __VU,
    k6_iteration: __ITER,
    test_type: 'single-endpoint',
    target_endpoint: TARGET_ENDPOINT,
    test_mode: Object.keys(endpointConfig.scenarios)[0],
  };

  const url = `${config.base.host}/v1/${TARGET_ENDPOINT}/insert`;
  const headers = getAuthHeaders();
  
  const startTime = Date.now();
  
  const response = http.post(url, JSON.stringify(payload), {
    headers: headers,
    timeout: config.base.timeout,
    tags: { 
      endpoint: TARGET_ENDPOINT,
      test_mode: 'single-endpoint',
    },
  });
  
  const endTime = Date.now();
  const duration = endTime - startTime;

  // Record endpoint-specific metrics
  endpointRequests.add(1);
  endpointDuration.add(duration);

  // Endpoint-specific validation
  const checks = {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'response has success field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hasOwnProperty('success');
      } catch {
        return false;
      }
    },
    'response time acceptable': (r) => r.timings.duration < 3000,
    'job_id returned': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success && body.data && body.data.job_id;
      } catch {
        return false;
      }
    },
  };

  // Add endpoint-specific checks
  if (TARGET_ENDPOINT === 'user-activities') {
    checks['user activities fast response'] = (r) => r.timings.duration < 800;
  } else if (TARGET_ENDPOINT === 'transaction-events') {
    checks['transaction secure headers'] = (r) => r.headers['X-Request-ID'] !== undefined;
  } else if (TARGET_ENDPOINT === 'security-events') {
    checks['security event processed'] = (r) => r.status < 400;
  } else if (TARGET_ENDPOINT === 'callback-logs') {
    checks['callback acknowledged'] = (r) => r.status < 300;
  }

  const checkResult = check(response, checks);
  endpointErrors.add(!checkResult);

  // Enhanced logging for single endpoint focus
  if (!checkResult || response.status >= 400) {
    console.error(`❌ ${TARGET_ENDPOINT} failed:`, {
      status: response.status,
      duration: duration,
      iteration: __ITER,
      body: response.body.substring(0, 200),
    });
  } else if (__ITER % 50 === 0) {
    // More frequent success logging for single endpoint
    console.log(`✅ ${TARGET_ENDPOINT} - ${__ITER} requests (${duration}ms avg)`);
  }

  // Use endpoint-specific sleep pattern
  sleep(endpointConfig.sleepDuration());
}

// Setup function
export function setup() {
  console.log(`🎯 Single Endpoint Load Test: ${TARGET_ENDPOINT}`);
  console.log(`🏥 Health check...`);
  
  const healthCheck = http.get(`${config.base.host}/v1/health/live`);
  
  if (healthCheck.status !== 200) {
    throw new Error(`❌ Health check failed: ${healthCheck.status}`);
  }
  
  console.log(`✅ Service ready for ${TARGET_ENDPOINT} testing`);
  console.log(`📊 Test configuration: ${Object.keys(endpointConfig.scenarios)[0]}`);
  
  return { 
    startTime: Date.now(),
    targetEndpoint: TARGET_ENDPOINT,
  };
}

// Teardown function
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  
  console.log(`\n🏁 Single Endpoint Test Completed: ${data.targetEndpoint}`);
  console.log(`⏱️  Duration: ${duration.toFixed(2)} seconds`);
  console.log(`📈 Check ${data.targetEndpoint}-specific metrics above`);
}