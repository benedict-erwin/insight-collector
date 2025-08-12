// Dynamic Payload Generators for Load Testing
import { randomString, randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Utility functions
const generateUUID = () => {
  return 'xxxx-xxxx-xxxx-xxxx'.replace(/[x]/g, () => {
    return (Math.random() * 16 | 0).toString(16);
  });
};

const generateTimestamp = () => {
  // Random timestamp within last 24 hours in format "2025-08-10T14:21:02Z"
  const now = new Date();
  const randomMs = randomIntBetween(0, 24 * 60 * 60 * 1000); // Last 24h
  return new Date(now.getTime() - randomMs).toISOString().replace(/\.\d{3}Z$/, 'Z');
};

const generateUnixTimestamp = () => {
  // Random timestamp within last 24 hours (Unix timestamp)
  const now = Date.now();
  const randomMs = randomIntBetween(0, 24 * 60 * 60 * 1000); // Last 24h
  return Math.floor((now - randomMs) / 1000);
};

const generateRequestID = () => {
  return `req-${Date.now()}-${randomString(8)}`;
};

const generateIP = () => {
  return `${randomIntBetween(1, 254)}.${randomIntBetween(1, 254)}.${randomIntBetween(1, 254)}.${randomIntBetween(1, 254)}`;
};

// Sample data pools
const sampleData = {
  userIds: ['user_001', 'user_002', 'user_003', 'user_004', 'user_005', 'user_006', 'user_007', 'user_008'],
  activityTypes: ['login', 'logout', 'purchase', 'transfer', 'deposit', 'withdrawal', 'view_balance', 'api_call'],
  categories: ['authentication', 'transaction', 'account', 'api', 'security', 'reporting'],
  statuses: ['success', 'failed', 'pending', 'timeout', 'error'],
  channels: ['web', 'mobile', 'api', 'webhook'],
  currencies: ['USD', 'EUR', 'GBP', 'BTC', 'ETH', 'ADA', 'DOT', 'SOL'],
  userAgents: [
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
    'Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1',
    'PostmanRuntime/7.28.0',
    'curl/7.68.0',
  ],
  endpoints: ['/api/v1/users', '/api/v1/transactions', '/api/v1/balance', '/api/v1/transfer', '/api/v1/deposit'],
  threatLevels: ['low', 'medium', 'high', 'critical'],
  eventTypes: ['login_attempt', 'suspicious_transaction', 'rate_limit_exceeded', 'invalid_auth', 'data_breach_attempt'],
};

// User Activities Payload Generator
export function generateUserActivitiesPayload() {
  const basePayload = {
    user_id: randomItem(sampleData.userIds),
    session_id: `sess_${randomString(16)}`,
    activity_type: randomItem(sampleData.activityTypes),
    category: randomItem(sampleData.categories),
    subcategory: randomItem(['primary', 'secondary', 'system']),
    status: randomItem(sampleData.statuses),
    channel: randomItem(sampleData.channels),
    endpoint_group: randomItem(['auth', 'transaction', 'account', 'admin']),
    method: randomItem(['GET', 'POST', 'PUT', 'DELETE']),
    risk_level: randomItem(['low', 'medium', 'high']),
    request_id: generateRequestID(),
    trace_id: `trace_${randomString(24)}`,
    duration_ms: randomIntBetween(50, 2000),
    response_code: randomItem([200, 201, 400, 401, 403, 404, 500]),
    request_size_bytes: randomIntBetween(100, 5000),
    response_size_bytes: randomIntBetween(200, 10000),
    ip_address: generateIP(),
    user_agent: randomItem(sampleData.userAgents),
    app_version: `v${randomIntBetween(1, 3)}.${randomIntBetween(0, 9)}.${randomIntBetween(0, 9)}`,
    referrer_url: `https://app.insight.com/${randomItem(['dashboard', 'transactions', 'reports', 'settings'])}`,
    endpoint: randomItem(sampleData.endpoints),
    details: {
      browser: randomItem(['Chrome', 'Firefox', 'Safari', 'Edge']),
      os: randomItem(['Windows', 'macOS', 'Linux', 'iOS', 'Android']),
      device_type: randomItem(['desktop', 'mobile', 'tablet']),
      location: {
        country: randomItem(['US', 'UK', 'DE', 'JP', 'AU']),
        city: randomItem(['New York', 'London', 'Berlin', 'Tokyo', 'Sydney']),
      },
    },
    time: generateTimestamp(),
  };

  // Add some random optional fields
  if (Math.random() > 0.7) {
    basePayload.additional_context = {
      feature_flags: [`flag_${randomString(6)}`],
      ab_test_group: randomItem(['A', 'B', 'control']),
    };
  }

  return basePayload;
}

// Transaction Events Payload Generator  
export function generateTransactionEventsPayload() {
  const transactionTypes = ['buy', 'sell', 'transfer', 'deposit', 'withdrawal', 'swap'];
  const transactionType = randomItem(transactionTypes);
  
  return {
    user_id: randomItem(sampleData.userIds),
    transaction_id: `txn_${randomString(20)}`,
    session_id: `sess_${randomString(16)}`,
    transaction_type: transactionType,
    status: randomItem(['pending', 'completed', 'failed', 'cancelled']),
    amount: Math.random() * 10000,
    currency: randomItem(sampleData.currencies),
    fee_amount: Math.random() * 100,
    fee_currency: randomItem(['USD', 'EUR']),
    exchange: randomItem(['binance', 'coinbase', 'kraken', 'gemini', 'ftx']),
    wallet_address: `0x${randomString(40)}`,
    blockchain: randomItem(['ethereum', 'bitcoin', 'cardano', 'polkadot', 'solana']),
    confirmation_count: randomIntBetween(0, 12),
    gas_fee: transactionType === 'swap' ? Math.random() * 50 : null,
    request_id: generateRequestID(),
    trace_id: `trace_${randomString(24)}`,
    ip_address: generateIP(),
    user_agent: randomItem(sampleData.userAgents),
    channel: randomItem(sampleData.channels),
    endpoint: randomItem(sampleData.endpoints),
    method: randomItem(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']),
    details: {
      market_price_usd: Math.random() * 50000,
      slippage: transactionType === 'swap' ? Math.random() * 5 : null,
      execution_time_ms: randomIntBetween(100, 5000),
      priority: randomItem(['low', 'medium', 'high']),
    },
    time: generateTimestamp(),
  };
}

// Security Events Payload Generator
export function generateSecurityEventsPayload() {
  const eventType = randomItem(sampleData.eventTypes);
  
  return {
    user_id: randomItem([...sampleData.userIds, null]), // Some events may not have user_id
    session_id: Math.random() > 0.3 ? `sess_${randomString(16)}` : null,
    event_type: eventType,
    severity: randomItem(['info', 'warning', 'error', 'critical']),
    threat_level: randomItem(sampleData.threatLevels),
    status: randomItem(['detected', 'blocked', 'allowed', 'investigating']),
    auth_stage: randomItem(['login', 'verification', 'authorization', 'session']),
    action_taken: randomItem(['block', 'alert', 'log', 'escalate', 'none']),
    ip_address: generateIP(),
    source_ip: generateIP(),
    target_ip: Math.random() > 0.5 ? generateIP() : null,
    user_agent: randomItem(sampleData.userAgents),
    endpoint: randomItem(sampleData.endpoints),
    method: randomItem(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']),
    response_code: randomItem([200, 400, 401, 403, 404, 429, 500, 503]),
    request_id: generateRequestID(),
    trace_id: `trace_${randomString(24)}`,
    geolocation: {
      country: randomItem(['US', 'CN', 'RU', 'BR', 'IN', 'UK', 'DE']),
      region: randomItem(['California', 'Beijing', 'Moscow', 'London']),
      city: randomItem(['Los Angeles', 'Shanghai', 'St. Petersburg', 'Manchester']),
      coordinates: `${(Math.random() * 180 - 90).toFixed(6)},${(Math.random() * 360 - 180).toFixed(6)}`,
    },
    details: {
      attack_vector: eventType === 'data_breach_attempt' ? randomItem(['sql_injection', 'xss', 'csrf', 'brute_force']) : null,
      failed_attempts: eventType.includes('login') ? randomIntBetween(1, 10) : null,
      blocked_reason: randomItem(['rate_limit', 'suspicious_pattern', 'geo_restriction', 'blacklist']),
      risk_score: randomIntBetween(1, 100),
      rule_triggered: `security_rule_${randomIntBetween(1, 50)}`,
    },
    time: generateTimestamp(),
  };
}

// Callback Logs Payload Generator
export function generateCallbackLogsPayload() {
  const callbackTypes = ['webhook', 'api_callback', 'payment_notification', 'order_update', 'transaction_confirmation'];
  const callbackType = randomItem(callbackTypes);
  
  return {
    callback_id: `cb_${randomString(24)}`,
    callback_type: callbackType,
    source_system: randomItem(['payment_gateway', 'exchange_api', 'blockchain_monitor', 'third_party_service']),
    destination_url: `https://api.insight.com/callbacks/${callbackType}`,
    target_endpoint: `https://api.insight.com/callbacks/${callbackType}`,
    times_attempted: randomIntBetween(1, 5),
    http_method: randomItem(['POST', 'PUT', 'PATCH']),
    status_code: randomItem([200, 201, 400, 401, 403, 404, 422, 500, 502, 503]),
    response_time_ms: randomIntBetween(50, 3000),
    retry_count: randomIntBetween(0, 3),
    payload_size_bytes: randomIntBetween(500, 50000),
    request_id: generateRequestID(),
    trace_id: `trace_${randomString(24)}`,
    user_agent: randomItem(['PaymentGateway/1.0', 'ExchangeAPI/2.1', 'BlockchainMonitor/3.0']),
    ip_address: generateIP(),
    payloads: {
      headers: {
        'Content-Type': 'application/json',
        'X-Signature': randomString(64),
        'X-Timestamp': Math.floor(Date.now() / 1000).toString(),
        'User-Agent': `CallbackService/${randomIntBetween(1, 5)}.${randomIntBetween(0, 9)}`,
      },
      body: generateCallbackBody(callbackType),
      query_params: {
        version: `v${randomIntBetween(1, 3)}`,
        format: 'json',
      },
    },
    details: {
      webhook_id: `wh_${randomString(20)}`,
      event_id: `evt_${randomString(16)}`,
      correlation_id: generateUUID(),
      processing_status: randomItem(['received', 'processing', 'completed', 'failed']),
    },
    time: generateTimestamp(),
  };
}

// Helper function for callback body generation
function generateCallbackBody(callbackType) {
  const bodies = {
    webhook: {
      event: randomItem(['user.created', 'transaction.completed', 'payment.failed']),
      data: {
        id: randomString(16),
        amount: (Math.random() * 1000).toFixed(2),
        currency: randomItem(sampleData.currencies),
      },
    },
    payment_notification: {
      payment_id: `pay_${randomString(20)}`,
      amount: (Math.random() * 5000).toFixed(2),
      currency: randomItem(['USD', 'EUR', 'GBP']),
      status: randomItem(['completed', 'failed', 'pending']),
    },
    transaction_confirmation: {
      txn_hash: `0x${randomString(64)}`,
      block_height: randomIntBetween(1000000, 2000000),
      confirmations: randomIntBetween(1, 12),
      amount: (Math.random() * 100).toFixed(8),
    },
  };

  return bodies[callbackType] || { type: callbackType, data: { id: randomString(12) } };
}

// Export all generators
export const payloadGenerators = {
  'user-activities': generateUserActivitiesPayload,
  'transaction-events': generateTransactionEventsPayload,
  'security-events': generateSecurityEventsPayload,  
  'callback-logs': generateCallbackLogsPayload,
};