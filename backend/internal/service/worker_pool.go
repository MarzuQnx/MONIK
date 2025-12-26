package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"monik-enterprise/internal/config"
)

// WorkerPool manages a pool of workers for concurrent monitoring
type WorkerPool struct {
	config         config.WorkerPoolConfig
	workers        []*Worker
	jobQueue       chan Job
	workerPool     chan chan Job
	quit           chan bool
	wg             sync.WaitGroup
	mu             sync.RWMutex
	metrics        *WorkerMetrics
	circuitBreaker *CircuitBreaker
	loadBalancer   *LoadBalancer
}

// Worker represents a worker in the pool
type Worker struct {
	ID         int
	JobQueue   chan Job
	WorkerPool chan chan Job
	Quit       chan bool
	Service    *MikroTikService
	ActiveJobs int
	Stats      *WorkerStats
}

// Job represents a monitoring job
type Job struct {
	InterfaceName string
	Type          string // traffic, stats, discovery
	Timeout       time.Duration
	RetryCount    int
	MaxRetries    int
	Priority      int // 0: low, 1: medium, 2: high
	CreatedAt     time.Time
}

// WorkerMetrics tracks worker pool performance
type WorkerMetrics struct {
	ActiveJobs   int64
	TotalJobs    int64
	SuccessJobs  int64
	FailedJobs   int64
	AvgResponse  time.Duration
	LastActivity time.Time
	WorkerStats  map[int]*WorkerStats
	mu           sync.RWMutex
}

// WorkerStats tracks individual worker performance
type WorkerStats struct {
	ActiveJobs   int
	TotalJobs    int64
	SuccessJobs  int64
	FailedJobs   int64
	AvgResponse  time.Duration
	LastActivity time.Time
	Errors       int64
	LastError    time.Time
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	state        CircuitState
	failureCount int64
	successCount int64
	lastFailure  time.Time
	mu           sync.RWMutex
	config       CircuitBreakerConfig
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	HalfOpenMaxCalls int
}

// LoadBalancer implements load balancing strategies
type LoadBalancer struct {
	strategy LoadBalancingStrategy
	mu       sync.RWMutex
	lastUsed int
}

// LoadBalancingStrategy defines load balancing strategies
type LoadBalancingStrategy int

const (
	RoundRobin LoadBalancingStrategy = iota
	LeastConnections
	Random
	WeightedRoundRobin
)

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config config.WorkerPoolConfig, service *MikroTikService) *WorkerPool {
	pool := &WorkerPool{
		config:     config,
		workers:    make([]*Worker, 0, config.MaxWorkers),
		jobQueue:   make(chan Job, config.QueueSize),
		workerPool: make(chan chan Job, config.MaxWorkers),
		quit:       make(chan bool),
		metrics: &WorkerMetrics{
			WorkerStats: make(map[int]*WorkerStats),
		},
		circuitBreaker: NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  60 * time.Second,
			HalfOpenMaxCalls: 3,
		}),
		loadBalancer: NewLoadBalancer(RoundRobin),
	}

	// Create workers
	for i := 0; i < config.MaxWorkers; i++ {
		worker := &Worker{
			ID:         i,
			JobQueue:   make(chan Job, 1),
			WorkerPool: pool.workerPool,
			Quit:       make(chan bool),
			Service:    service,
			Stats: &WorkerStats{
				LastActivity: time.Now(),
			},
		}
		pool.workers = append(pool.workers, worker)
		pool.metrics.WorkerStats[i] = worker.Stats
	}

	return pool
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:  CircuitClosed,
		config: config,
	}
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(strategy LoadBalancingStrategy) *LoadBalancer {
	return &LoadBalancer{
		strategy: strategy,
		lastUsed: 0,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	wp.wg.Add(len(wp.workers))

	// Start workers
	for _, worker := range wp.workers {
		go wp.startWorker(worker)
	}

	// Start dispatcher
	go wp.dispatch()

	// Start circuit breaker monitoring
	go wp.circuitBreaker.monitor()
}

// monitor monitors the circuit breaker state
func (cb *CircuitBreaker) monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// S1000 Fix: Menggunakan for range ticker
	for range ticker.C {
		cb.checkState()
	}
}

// checkState checks and updates the circuit breaker state
func (cb *CircuitBreaker) checkState() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.config.RecoveryTimeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
		}
	case CircuitHalfOpen:
		if cb.successCount >= int64(cb.config.HalfOpenMaxCalls) {
			cb.state = CircuitClosed
			cb.failureCount = 0
		}
	}
}

// Allow checks if a request should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state != CircuitOpen
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.successCount++
	}
	cb.failureCount = 0
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.failureCount >= int64(cb.config.FailureThreshold) {
		cb.state = CircuitOpen
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// SelectWorker selects the best worker based on load balancing strategy
func (lb *LoadBalancer) SelectWorker(workers []*Worker) *Worker {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(workers) == 0 {
		return nil
	}

	switch lb.strategy {
	case RoundRobin:
		worker := workers[lb.lastUsed%len(workers)]
		lb.lastUsed++
		return worker
	case LeastConnections:
		return lb.selectLeastConnections(workers)
	case Random:
		return workers[time.Now().Nanosecond()%len(workers)]
	case WeightedRoundRobin:
		return lb.selectWeightedRoundRobin(workers)
	default:
		return workers[0]
	}
}

// selectLeastConnections selects the worker with the least active connections
func (lb *LoadBalancer) selectLeastConnections(workers []*Worker) *Worker {
	var selected *Worker
	minConnections := int(^uint(0) >> 1) // Max int

	for _, worker := range workers {
		if worker.ActiveJobs < minConnections {
			minConnections = worker.ActiveJobs
			selected = worker
		}
	}

	return selected
}

// selectWeightedRoundRobin selects workers based on their performance weights
func (lb *LoadBalancer) selectWeightedRoundRobin(workers []*Worker) *Worker {
	// Calculate weights based on success rate
	totalWeight := 0
	weights := make([]int, len(workers))

	for i, worker := range workers {
		// Calculate success rate (avoid division by zero)
		totalJobs := worker.Stats.TotalJobs
		if totalJobs == 0 {
			weights[i] = 100 // Default weight
		} else {
			successRate := float64(worker.Stats.SuccessJobs) / float64(totalJobs)
			weights[i] = int(successRate * 100)
		}
		totalWeight += weights[i]
	}

	// Select based on weight
	if totalWeight == 0 {
		return workers[0]
	}

	randVal := time.Now().Nanosecond() % totalWeight
	currentWeight := 0

	for i, weight := range weights {
		currentWeight += weight
		if randVal < currentWeight {
			return workers[i]
		}
	}

	return workers[0]
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.quit)

	// Stop all workers
	for _, worker := range wp.workers {
		close(worker.Quit)
	}

	wp.wg.Wait()
}

// SubmitJob submits a job to the worker pool
func (wp *WorkerPool) SubmitJob(job Job) error {
	select {
	case wp.jobQueue <- job:
		wp.metrics.mu.Lock()
		wp.metrics.TotalJobs++
		wp.metrics.mu.Unlock()
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("job queue is full")
	}
}

// GetMetrics returns worker pool metrics
func (wp *WorkerPool) GetMetrics() *WorkerMetrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	metrics := &WorkerMetrics{
		ActiveJobs:   wp.metrics.ActiveJobs,
		TotalJobs:    wp.metrics.TotalJobs,
		SuccessJobs:  wp.metrics.SuccessJobs,
		FailedJobs:   wp.metrics.FailedJobs,
		AvgResponse:  wp.metrics.AvgResponse,
		LastActivity: wp.metrics.LastActivity,
		WorkerStats:  make(map[int]*WorkerStats),
	}

	for id, stats := range wp.metrics.WorkerStats {
		metrics.WorkerStats[id] = &WorkerStats{
			ActiveJobs:   stats.ActiveJobs,
			TotalJobs:    stats.TotalJobs,
			SuccessJobs:  stats.SuccessJobs,
			FailedJobs:   stats.FailedJobs,
			AvgResponse:  stats.AvgResponse,
			LastActivity: stats.LastActivity,
		}
	}

	return metrics
}

// dispatch distributes jobs to workers with load balancing and circuit breaker
func (wp *WorkerPool) dispatch() {
	for {
		select {
		case job := <-wp.jobQueue:
			// Check circuit breaker
			if !wp.circuitBreaker.Allow() {
				// Circuit is open, drop job or implement fallback
				wp.metrics.mu.Lock()
				wp.metrics.FailedJobs++
				wp.metrics.mu.Unlock()
				continue
			}

			// Select worker using load balancer
			worker := wp.loadBalancer.SelectWorker(wp.workers)
			if worker == nil {
				// No workers available, put job back in queue
				wp.jobQueue <- job
				continue
			}

			// Submit job to worker
			select {
			case worker.JobQueue <- job:
				worker.ActiveJobs++
				wp.metrics.mu.Lock()
				wp.metrics.ActiveJobs++
				wp.metrics.LastActivity = time.Now()
				wp.metrics.mu.Unlock()
			case <-time.After(time.Second):
				// Worker is busy, put job back in queue
				wp.jobQueue <- job
			}
		case <-wp.quit:
			return
		}
	}
}

// startWorker starts a worker
func (wp *WorkerPool) startWorker(worker *Worker) {
	defer wp.wg.Done()

	for {
		// Register worker in pool
		wp.workerPool <- worker.JobQueue

		select {
		case job := <-worker.JobQueue:
			wp.processJob(worker, job)
		case <-worker.Quit:
			return
		}
	}
}

// processJob processes a job with exponential backoff and circuit breaker
func (wp *WorkerPool) processJob(worker *Worker, job Job) {
	startTime := time.Now()

	// Update worker stats
	wp.metrics.mu.Lock()
	stats := wp.metrics.WorkerStats[worker.ID]
	stats.ActiveJobs++
	stats.TotalJobs++
	stats.LastActivity = time.Now()
	wp.metrics.mu.Unlock()

	defer func() {
		// Update worker stats
		wp.metrics.mu.Lock()
		stats.ActiveJobs--
		duration := time.Since(startTime)
		stats.AvgResponse = (stats.AvgResponse*time.Duration(stats.TotalJobs-1) + duration) / time.Duration(stats.TotalJobs)
		wp.metrics.ActiveJobs--
		wp.metrics.mu.Unlock()
	}()

	// Process job based on type
	var err error
	switch job.Type {
	case "traffic":
		_, err = worker.Service.GetTrafficStats(context.Background(), job.InterfaceName)
	case "stats":
		_, err = worker.Service.GetTrafficStats(context.Background(), job.InterfaceName)
	case "discovery":
		_, err = worker.Service.GetInterfaces(context.Background())
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Handle job result
	if err != nil {
		wp.metrics.mu.Lock()
		wp.metrics.FailedJobs++
		stats.FailedJobs++
		stats.Errors++
		stats.LastError = time.Now()
		wp.metrics.mu.Unlock()

		// Record failure in circuit breaker
		wp.circuitBreaker.RecordFailure()

		// Exponential backoff retry logic
		if job.RetryCount < job.MaxRetries {
			job.RetryCount++
			backoffDuration := time.Duration(1<<uint(job.RetryCount)) * time.Second
			if backoffDuration > 30*time.Second {
				backoffDuration = 30 * time.Second
			}
			time.Sleep(backoffDuration)
			wp.SubmitJob(job)
		}
	} else {
		wp.metrics.mu.Lock()
		wp.metrics.SuccessJobs++
		stats.SuccessJobs++
		wp.metrics.mu.Unlock()

		// Record success in circuit breaker
		wp.circuitBreaker.RecordSuccess()
	}
}

// GetLoad returns the current load of the worker pool
func (wp *WorkerPool) GetLoad() float64 {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	if wp.metrics.TotalJobs == 0 {
		return 0
	}

	return float64(wp.metrics.ActiveJobs) / float64(wp.config.MaxWorkers)
}

// ShouldRebalance checks if rebalancing is needed
func (wp *WorkerPool) ShouldRebalance() bool {
	load := wp.GetLoad()
	return load > wp.config.LoadThreshold
}

// Rebalance redistributes work among workers
func (wp *WorkerPool) Rebalance() {
	// This is a placeholder for rebalancing logic
	// In a real implementation, this would redistribute jobs
	// based on worker performance and load
}

// GetWorkerCount returns the number of active workers
func (wp *WorkerPool) GetWorkerCount() int {
	return len(wp.workers)
}

// GetQueueSize returns the current queue size
func (wp *WorkerPool) GetQueueSize() int {
	return len(wp.jobQueue)
}

// GetQueueCapacity returns the queue capacity
func (wp *WorkerPool) GetQueueCapacity() int {
	return cap(wp.jobQueue)
}
