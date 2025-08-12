// K6 Load Testing Configuration
export const config = {
  // Base configuration
  base: {
    host: __ENV.TEST_HOST || 'http://localhost:8080',
    timeout: '30s',
  },

  // Authentication - For non-auth testing, use dummy token
  auth: {
    jwt_token: __ENV.JWT_TOKEN || 'dummy-token-for-non-auth-testing',
  },

  // Test scenarios configuration  
  scenarios: {
    // Smoke test - Basic functionality
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
      tags: { test_type: 'smoke' },
    },

    // Load test - Normal load
    load: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '2m', target: 10 },   // Ramp up to 10 users
        { duration: '5m', target: 10 },   // Stay at 10 users  
        { duration: '2m', target: 0 },    // Ramp down
      ],
      tags: { test_type: 'load' },
    },

    // Stress test - High load
    stress: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '2m', target: 20 },   // Ramp up to 20 users
        { duration: '5m', target: 20 },   // Stay at 20 users
        { duration: '2m', target: 50 },   // Ramp up to 50 users
        { duration: '5m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 0 },    // Ramp down
      ],
      tags: { test_type: 'stress' },
    },

    // Spike test - Sudden traffic spikes  
    spike: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '10s', target: 1 },   // Normal traffic
        { duration: '10s', target: 250 }, // Spike to 250 users
        { duration: '30s', target: 250 }, // Stay at spike
        { duration: '10s', target: 1 },   // Return to normal
      ],
      tags: { test_type: 'spike' },
    },

    // Mixed load test - Heavy realistic production traffic
    mixed: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: '1m', target: 30 },   // Quick startup (30 users)
        { duration: '3m', target: 80 },   // Business hours buildup (80 users ~40-50 RPS)
        { duration: '5m', target: 150 },  // Peak business hours (150 users ~75-90 RPS)
        { duration: '3m', target: 200 },  // Peak load test (200 users ~100-120 RPS)
        { duration: '4m', target: 120 },  // Afternoon moderate (120 users ~60-80 RPS)
        { duration: '2m', target: 60 },   // Evening hours (60 users ~30-40 RPS)
        { duration: '2m', target: 20 },   // Night time (20 users ~10-15 RPS)
        { duration: '1m', target: 0 },    // Graceful shutdown
      ],
      tags: { test_type: 'mixed' },
    },
  },

  // Performance thresholds
  thresholds: {
    // HTTP response time thresholds
    'http_req_duration': ['p(95)<2000'], // 95% of requests under 2s
    'http_req_duration{expected_response:true}': ['p(95)<1500'], // Success requests under 1.5s
    
    // HTTP failure rate thresholds  
    'http_req_failed': ['rate<0.05'], // Less than 5% failure rate
    
    // Per-endpoint thresholds
    'http_req_duration{endpoint:user-activities}': ['p(95)<1000'],
    'http_req_duration{endpoint:transaction-events}': ['p(95)<1000'], 
    'http_req_duration{endpoint:security-events}': ['p(95)<1000'],
    'http_req_duration{endpoint:callback-logs}': ['p(95)<1000'],

    // Queue processing thresholds (custom metrics)
    'job_enqueue_duration': ['p(95)<100'], // Job enqueue under 100ms
  },
};

// Environment-specific overrides
export function getConfig(testType = 'load') {
  const selectedScenario = {};
  selectedScenario[testType] = config.scenarios[testType];
  
  return {
    scenarios: selectedScenario,
    thresholds: config.thresholds,
  };
}