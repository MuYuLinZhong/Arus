package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path", "status"},
	)

	LockChallengeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lock_challenge_total",
			Help: "Total number of challenge requests",
		},
		[]string{"result"},
	)

	ActiveSessionsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_sessions_total",
			Help: "Current number of active sessions",
		},
	)

	DBPoolOpenConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_open_connections",
			Help: "Number of open database connections",
		},
	)

	MQQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mq_queue_depth",
			Help: "Number of messages in RabbitMQ queues",
		},
		[]string{"queue"},
	)
)

func Init() {
	prometheus.MustRegister(
		HTTPRequestDuration,
		LockChallengeTotal,
		ActiveSessionsTotal,
		DBPoolOpenConnections,
		MQQueueDepth,
	)
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// 继续下一个中间件/处理函数,暂时放权,处理完之后继续来这里;
		c.Next()
		duration := time.Since(start).Seconds()

		HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			strconv.Itoa(c.Writer.Status()),
		).Observe(duration)
	}
}

func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
