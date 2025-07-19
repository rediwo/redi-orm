package mcp

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	// Authentication
	EnableAuth   bool
	APIKey       string
	AllowedHosts []string
	
	// Rate limiting
	EnableRateLimit bool
	RequestsPerMin  int
	BurstLimit      int
	
	// Permissions
	AllowedTables    []string
	ForbiddenTables  []string
	ReadOnlyMode     bool
	MaxQueryRows     int
	QueryTimeout     time.Duration
	
	// SQL validation
	AllowedPatterns  []string
	ForbiddenQueries []string
}

// SecurityManager handles authentication, authorization, and rate limiting
type SecurityManager struct {
	config      SecurityConfig
	rateLimiter *RateLimiter
	mu          sync.RWMutex
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config SecurityConfig) *SecurityManager {
	sm := &SecurityManager{
		config: config,
	}
	
	if config.EnableRateLimit {
		sm.rateLimiter = NewRateLimiter(config.RequestsPerMin, config.BurstLimit)
	}
	
	return sm
}

// AuthenticateRequest validates authentication for HTTP requests
func (sm *SecurityManager) AuthenticateRequest(r *http.Request) error {
	if !sm.config.EnableAuth {
		return nil
	}
	
	// Check API key
	apiKey := r.Header.Get("Authorization")
	if apiKey == "" {
		apiKey = r.Header.Get("X-API-Key")
	}
	
	// Remove Bearer prefix if present
	if strings.HasPrefix(apiKey, "Bearer ") {
		apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	}
	
	if apiKey == "" {
		return fmt.Errorf("missing API key")
	}
	
	// Use constant time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(sm.config.APIKey)) != 1 {
		return fmt.Errorf("invalid API key")
	}
	
	return nil
}

// ValidateHost checks if the request comes from an allowed host
func (sm *SecurityManager) ValidateHost(r *http.Request) error {
	if len(sm.config.AllowedHosts) == 0 {
		return nil // No host restriction
	}
	
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	
	for _, allowedHost := range sm.config.AllowedHosts {
		if host == allowedHost || strings.HasSuffix(host, "."+allowedHost) {
			return nil
		}
	}
	
	return fmt.Errorf("host not allowed: %s", host)
}

// CheckRateLimit verifies if the request is within rate limits
func (sm *SecurityManager) CheckRateLimit(clientIP string) error {
	if !sm.config.EnableRateLimit || sm.rateLimiter == nil {
		return nil
	}
	
	if !sm.rateLimiter.Allow(clientIP) {
		return fmt.Errorf("rate limit exceeded")
	}
	
	return nil
}

// ValidateTableAccess checks if access to a table is allowed
func (sm *SecurityManager) ValidateTableAccess(tableName string) error {
	// Check forbidden tables first
	for _, forbidden := range sm.config.ForbiddenTables {
		if tableName == forbidden {
			return fmt.Errorf("access to table '%s' is forbidden", tableName)
		}
	}
	
	// If allowed tables are specified, check if table is in the list
	if len(sm.config.AllowedTables) > 0 {
		for _, allowed := range sm.config.AllowedTables {
			if tableName == allowed {
				return nil
			}
		}
		return fmt.Errorf("access to table '%s' is not allowed", tableName)
	}
	
	return nil
}

// ValidateQuery performs SQL query validation
func (sm *SecurityManager) ValidateQuery(sql string) error {
	sql = strings.TrimSpace(strings.ToUpper(sql))
	
	// Check forbidden queries
	for _, forbidden := range sm.config.ForbiddenQueries {
		if strings.Contains(sql, strings.ToUpper(forbidden)) {
			return fmt.Errorf("forbidden query pattern: %s", forbidden)
		}
	}
	
	// In read-only mode, only allow SELECT queries
	if sm.config.ReadOnlyMode {
		if !strings.HasPrefix(sql, "SELECT") {
			return fmt.Errorf("only SELECT queries are allowed in read-only mode")
		}
	}
	
	// Check for dangerous patterns
	dangerousPatterns := []string{
		"DROP TABLE", "DROP DATABASE", "DROP SCHEMA",
		"TRUNCATE", "DELETE FROM", "UPDATE SET",
		"INSERT INTO", "CREATE TABLE", "ALTER TABLE",
		"EXEC ", "EXECUTE ", "xp_", "sp_",
		"INTO OUTFILE", "INTO DUMPFILE", "LOAD_FILE",
		"--", "/*", "*/", ";--", "UNION SELECT",
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(sql, pattern) {
			return fmt.Errorf("potentially dangerous query pattern detected: %s", pattern)
		}
	}
	
	// Check allowed patterns if specified
	if len(sm.config.AllowedPatterns) > 0 {
		allowed := false
		for _, pattern := range sm.config.AllowedPatterns {
			if strings.Contains(sql, strings.ToUpper(pattern)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("query does not match allowed patterns")
		}
	}
	
	return nil
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu          sync.RWMutex
	clients     map[string]*ClientBucket
	limit       int
	burst       int
	cleanupTick *time.Ticker
}

// ClientBucket represents a token bucket for a specific client
type ClientBucket struct {
	tokens   int
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMin, burstLimit int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ClientBucket),
		limit:   requestsPerMin,
		burst:   burstLimit,
	}
	
	// Start cleanup routine
	rl.cleanupTick = time.NewTicker(5 * time.Minute)
	go rl.cleanup()
	
	return rl
}

// Allow checks if a request from the given client IP is allowed
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Get or create client bucket
	bucket, exists := rl.clients[clientIP]
	if !exists {
		bucket = &ClientBucket{
			tokens:   rl.burst,
			lastSeen: now,
		}
		rl.clients[clientIP] = bucket
	}
	
	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(bucket.lastSeen)
	tokensToAdd := int(elapsed.Minutes() * float64(rl.limit))
	
	bucket.tokens += tokensToAdd
	if bucket.tokens > rl.burst {
		bucket.tokens = rl.burst
	}
	bucket.lastSeen = now
	
	// Check if request is allowed
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// cleanup removes old client buckets
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTick.C {
		rl.mu.Lock()
		now := time.Now()
		for clientIP, bucket := range rl.clients {
			if now.Sub(bucket.lastSeen) > 10*time.Minute {
				delete(rl.clients, clientIP)
			}
		}
		rl.mu.Unlock()
	}
}

// Stop stops the rate limiter cleanup routine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTick != nil {
		rl.cleanupTick.Stop()
	}
}

// SecurityMiddleware returns an HTTP middleware for security validation
func (sm *SecurityManager) SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}
		if real := r.Header.Get("X-Real-IP"); real != "" {
			clientIP = real
		}
		
		// Validate host
		if err := sm.ValidateHost(r); err != nil {
			http.Error(w, "Forbidden: "+err.Error(), http.StatusForbidden)
			return
		}
		
		// Check rate limit
		if err := sm.CheckRateLimit(clientIP); err != nil {
			http.Error(w, "Too Many Requests: "+err.Error(), http.StatusTooManyRequests)
			return
		}
		
		// Authenticate request
		if err := sm.AuthenticateRequest(r); err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// GetStats returns security statistics
func (sm *SecurityManager) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"auth_enabled":      sm.config.EnableAuth,
		"rate_limit_enabled": sm.config.EnableRateLimit,
		"read_only_mode":    sm.config.ReadOnlyMode,
		"max_query_rows":    sm.config.MaxQueryRows,
	}
	
	if sm.rateLimiter != nil {
		sm.rateLimiter.mu.RLock()
		stats["active_clients"] = len(sm.rateLimiter.clients)
		sm.rateLimiter.mu.RUnlock()
	}
	
	return stats
}