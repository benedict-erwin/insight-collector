import http from 'k6/http';
import { check, sleep } from 'k6';

// ISOLATED CONFIG - No shared imports that might cause side effects
const TEST_HOST = __ENV.TEST_HOST || 'http://localhost:8080';

export const options = {
  scenarios: {
    health_only_isolated: {
      executor: 'ramping-vus',
      stages: [
        { duration: '1m', target: 20 },   // Ramp up to 20 users
        { duration: '1m', target: 50 },   // Ramp up to 50 users  
        { duration: '1m', target: 100 },  // Ramp up to 100 users
        { duration: '2m', target: 150 },  // Stay at 150 users
        { duration: '2m', target: 250 },  // Ramp down to 250 users
        { duration: '2m', target: 50 },   // Ramp down to 50 users
        { duration: '2m', target: 20 },   // Ramp down to 20 users
        { duration: '1m', target: 0 },    // Final ramp down
      ],
      startVUs: 1,
      tags: { test_type: 'health_only_isolated' },
    }
  },
  thresholds: {
    http_req_duration: ['p(95)<1000'],     // Health should be fast
    http_req_failed: ['rate<0.01'],        // Very low failure rate
    'http_req_duration{endpoint:health-live}': ['p(95)<500'],
    'http_req_duration{endpoint:health-ready}': ['p(95)<500'],
  },
  // Optimize metrics collection for high load
  summaryTrendStats: ['avg', 'p(95)', 'p(99)', 'max'],
  summaryTimeUnit: 's',
};

export default function () {
  // ABSOLUTELY ONLY health endpoints - hardcoded to prevent any config interference
  const endpoints = [
    `${TEST_HOST}/v1/health/live`,     
    `${TEST_HOST}/v1/health/ready`,    
  ];
  
  // Random selection
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  
  // Debug logging for first few iterations
  if (__VU == 1 && __ITER < 3) {
    console.log(`ISOLATED Health test hitting: ${endpoint}`);
  }
  
  const response = http.get(endpoint, {
    timeout: '30s',
    tags: { 
      endpoint: endpoint.includes('live') ? 'health-live' : 'health-ready',
      test_type: 'health_only_isolated'
    }
  });

  // Validation
  check(response, {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'response has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success !== undefined || body.data !== undefined;
      } catch (e) {
        console.error(`Failed to parse response: ${r.body}`);
        return false;
      }
    },
    'response time < 500ms': (r) => r.timings.duration < 500,
    'no timeout': (r) => r.timings.duration < 25000, // Less than 25s = no timeout
  });

  // Brief pause
  sleep(0.1);
}