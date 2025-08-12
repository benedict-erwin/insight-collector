// Performance Monitoring Integration for K6 Load Tests
import http from 'k6/http';
import { check } from 'k6';

// Real-time performance monitoring during load tests
export class PerformanceMonitor {
  constructor(config = {}) {
    this.config = {
      host: config.host || 'http://localhost:8080',
      monitoringInterval: config.monitoringInterval || 10000, // 10 seconds
      thresholds: config.thresholds || this.getDefaultThresholds(),
      alertWebhook: config.alertWebhook || null,
      ...config,
    };
    
    this.metrics = {
      healthChecks: [],
      workerStats: [],
      queueStats: [],
      systemMetrics: [],
    };
    
    this.alertsSent = new Set();
  }

  // Default performance thresholds
  getDefaultThresholds() {
    return {
      healthScore: 90,           // Minimum health score %
      responseTime: 2000,        // Max response time in ms
      errorRate: 5,             // Max error rate %
      queueBacklog: 1000,       // Max queue backlog
      workerUtilization: 85,    // Max worker utilization %
      memoryUsage: 80,          // Max memory usage %
      cpuUsage: 75,             // Max CPU usage %
    };
  }

  // Start monitoring during load test
  async startMonitoring() {
    console.log('🔍 Starting performance monitoring...');
    
    const monitoringLoop = async () => {
      try {
        await this.collectHealthMetrics();
        await this.collectWorkerMetrics();
        await this.collectQueueMetrics();
        await this.analyzePerformance();
      } catch (error) {
        console.error('❌ Monitoring error:', error.message);
      }
    };

    // Start monitoring loop
    const intervalId = setInterval(monitoringLoop, this.config.monitoringInterval);
    
    // Return cleanup function
    return () => {
      clearInterval(intervalId);
      console.log('🔍 Performance monitoring stopped');
    };
  }

  // Collect health metrics from service
  async collectHealthMetrics() {
    const response = http.get(`${this.config.host}/v1/health`, {
      timeout: '5s',
      tags: { monitoring: 'health' },
    });

    const isHealthy = check(response, {
      'health endpoint accessible': (r) => r.status === 200,
      'health response valid': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success && body.data;
        } catch {
          return false;
        }
      },
    });

    if (isHealthy && response.status === 200) {
      try {
        const healthData = JSON.parse(response.body);
        this.metrics.healthChecks.push({
          timestamp: Date.now(),
          status: healthData.data.status,
          services: healthData.data.services || {},
          responseTime: response.timings.duration,
        });

        // Keep only last 50 health checks
        if (this.metrics.healthChecks.length > 50) {
          this.metrics.healthChecks = this.metrics.healthChecks.slice(-50);
        }
      } catch (error) {
        console.error('❌ Failed to parse health response:', error.message);
      }
    }
  }

  // Monitor worker performance and queue status
  async collectWorkerMetrics() {
    // This would typically call your worker status endpoint
    // For now, simulating based on your CLI commands
    try {
      // Simulate calling worker status (in real implementation, add API endpoint)
      const workerMetrics = {
        timestamp: Date.now(),
        activeJobs: Math.floor(Math.random() * 50), // Simulated
        completedJobs: Math.floor(Math.random() * 1000),
        failedJobs: Math.floor(Math.random() * 10),
        workerUtilization: Math.random() * 100,
        avgProcessingTime: Math.random() * 500 + 100,
      };

      this.metrics.workerStats.push(workerMetrics);
      
      // Keep only last 20 measurements
      if (this.metrics.workerStats.length > 20) {
        this.metrics.workerStats = this.metrics.workerStats.slice(-20);
      }
    } catch (error) {
      console.error('❌ Failed to collect worker metrics:', error.message);
    }
  }

  // Monitor queue backlog and processing rates
  async collectQueueMetrics() {
    try {
      // Simulate queue metrics (in production, integrate with Redis monitoring)
      const queueMetrics = {
        timestamp: Date.now(),
        queues: {
          critical: Math.floor(Math.random() * 100),
          default: Math.floor(Math.random() * 200),
          low: Math.floor(Math.random() * 300),
        },
        totalBacklog: 0,
        processingRate: Math.random() * 50 + 10, // jobs per second
      };

      queueMetrics.totalBacklog = Object.values(queueMetrics.queues).reduce((sum, count) => sum + count, 0);
      
      this.metrics.queueStats.push(queueMetrics);
      
      // Keep only last 30 measurements
      if (this.metrics.queueStats.length > 30) {
        this.metrics.queueStats = this.metrics.queueStats.slice(-30);
      }
    } catch (error) {
      console.error('❌ Failed to collect queue metrics:', error.message);
    }
  }

  // Analyze performance and trigger alerts
  async analyzePerformance() {
    const analysis = this.getCurrentPerformanceAnalysis();
    
    // Check thresholds and trigger alerts
    for (const [metric, value] of Object.entries(analysis.currentMetrics)) {
      const threshold = this.config.thresholds[metric];
      
      if (threshold && value > threshold) {
        await this.triggerAlert(metric, value, threshold);
      }
    }

    // Log performance summary every 5 measurements
    if (this.metrics.healthChecks.length % 5 === 0) {
      this.logPerformanceSummary(analysis);
    }
  }

  // Get current performance analysis
  getCurrentPerformanceAnalysis() {
    const latest = {
      health: this.metrics.healthChecks[this.metrics.healthChecks.length - 1],
      worker: this.metrics.workerStats[this.metrics.workerStats.length - 1],
      queue: this.metrics.queueStats[this.metrics.queueStats.length - 1],
    };

    return {
      timestamp: Date.now(),
      currentMetrics: {
        responseTime: latest.health?.responseTime || 0,
        workerUtilization: latest.worker?.workerUtilization || 0,
        queueBacklog: latest.queue?.totalBacklog || 0,
        processingRate: latest.queue?.processingRate || 0,
      },
      trends: this.calculateTrends(),
      healthScore: this.calculateHealthScore(),
    };
  }

  // Calculate performance trends
  calculateTrends() {
    if (this.metrics.healthChecks.length < 5) return {};

    const recent = this.metrics.healthChecks.slice(-5);
    const responseTimes = recent.map(h => h.responseTime);
    
    return {
      responseTimetrend: this.calculateTrend(responseTimes),
      averageResponseTime: responseTimes.reduce((sum, time) => sum + time, 0) / responseTimes.length,
      improvementNeeded: responseTimes[responseTimes.length - 1] > responseTimes[0] * 1.2,
    };
  }

  // Calculate simple trend (improving/degrading)
  calculateTrend(values) {
    if (values.length < 2) return 'stable';
    
    const first = values[0];
    const last = values[values.length - 1];
    const change = (last - first) / first;
    
    if (change > 0.1) return 'degrading';
    if (change < -0.1) return 'improving';
    return 'stable';
  }

  // Calculate overall health score
  calculateHealthScore() {
    if (!this.metrics.healthChecks.length) return 0;
    
    const latest = this.metrics.healthChecks[this.metrics.healthChecks.length - 1];
    let score = 100;
    
    // Deduct points based on performance
    if (latest.responseTime > 1000) score -= 10;
    if (latest.responseTime > 2000) score -= 20;
    if (latest.status !== 'healthy') score -= 30;
    
    return Math.max(score, 0);
  }

  // Trigger performance alerts
  async triggerAlert(metric, currentValue, threshold) {
    const alertKey = `${metric}_${Math.floor(Date.now() / 60000)}`; // Alert once per minute per metric
    
    if (this.alertsSent.has(alertKey)) return;
    
    const alertMessage = {
      severity: 'warning',
      metric: metric,
      currentValue: currentValue,
      threshold: threshold,
      timestamp: new Date().toISOString(),
      message: `⚠️ Performance Alert: ${metric} is ${currentValue.toFixed(2)} (threshold: ${threshold})`,
    };

    console.warn(alertMessage.message);
    
    // Send webhook alert if configured
    if (this.config.alertWebhook) {
      try {
        http.post(this.config.alertWebhook, JSON.stringify(alertMessage), {
          headers: { 'Content-Type': 'application/json' },
          timeout: '5s',
        });
      } catch (error) {
        console.error('❌ Failed to send alert webhook:', error.message);
      }
    }
    
    this.alertsSent.add(alertKey);
  }

  // Log performance summary
  logPerformanceSummary(analysis) {
    console.log('\n📊 Performance Summary:');
    console.log(`   • Health Score: ${analysis.healthScore.toFixed(1)}%`);
    console.log(`   • Response Time: ${analysis.currentMetrics.responseTime.toFixed(0)}ms`);
    console.log(`   • Worker Utilization: ${analysis.currentMetrics.workerUtilization.toFixed(1)}%`);
    console.log(`   • Queue Backlog: ${analysis.currentMetrics.queueBacklog} jobs`);
    console.log(`   • Processing Rate: ${analysis.currentMetrics.processingRate.toFixed(1)} jobs/sec`);
    
    if (analysis.trends.responseTimetrend === 'degrading') {
      console.log('   ⚠️  Response time trend: DEGRADING');
    } else if (analysis.trends.responseTimetrend === 'improving') {
      console.log('   ✅ Response time trend: IMPROVING');
    }
    
    console.log('');
  }

  // Get final performance report
  generateFinalReport() {
    const totalHealthChecks = this.metrics.healthChecks.length;
    const totalWorkerStats = this.metrics.workerStats.length;
    
    if (totalHealthChecks === 0) {
      return { error: 'No performance data collected' };
    }

    const responseTimes = this.metrics.healthChecks.map(h => h.responseTime);
    const healthScores = this.metrics.healthChecks.map(h => h.status === 'healthy' ? 100 : 0);

    return {
      summary: {
        totalMeasurements: totalHealthChecks,
        testDuration: totalHealthChecks * (this.config.monitoringInterval / 1000), // seconds
        averageResponseTime: responseTimes.reduce((sum, time) => sum + time, 0) / responseTimes.length,
        maxResponseTime: Math.max(...responseTimes),
        minResponseTime: Math.min(...responseTimes),
        averageHealthScore: healthScores.reduce((sum, score) => sum + score, 0) / healthScores.length,
        uptimePercentage: (healthScores.filter(score => score === 100).length / healthScores.length) * 100,
      },
      recommendations: this.generateRecommendations(),
    };
  }

  // Generate performance recommendations
  generateRecommendations() {
    const report = [];
    const latest = this.getCurrentPerformanceAnalysis();
    
    if (latest.currentMetrics.responseTime > this.config.thresholds.responseTime) {
      report.push('🔧 Consider increasing worker concurrency or optimizing database queries');
    }
    
    if (latest.currentMetrics.queueBacklog > this.config.thresholds.queueBacklog) {
      report.push('🔧 Queue backlog is high - consider scaling workers or optimizing job processing');
    }
    
    if (latest.currentMetrics.workerUtilization > this.config.thresholds.workerUtilization) {
      report.push('🔧 Worker utilization is high - consider adding more workers or optimizing job handlers');
    }
    
    if (latest.healthScore < this.config.thresholds.healthScore) {
      report.push('🔧 Overall health score is low - investigate service dependencies');
    }
    
    if (report.length === 0) {
      report.push('✅ Performance looks good! No immediate optimizations needed');
    }
    
    return report;
  }
}

// Export singleton instance
export const performanceMonitor = new PerformanceMonitor();