#!/bin/bash
#
# Secure Production MCP Setup
# 
# This script demonstrates how to set up MCP in a production environment
# with full security features enabled.

# Configuration
MCP_PORT=8443
MCP_LOG_DIR="/var/log/redi-orm"
MCP_PID_FILE="/var/run/redi-orm-mcp.pid"
MCP_CONFIG_DIR="/etc/redi-orm"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root (recommended for production setup)
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script should be run as root for production setup"
        exit 1
    fi
}

# Create necessary directories
setup_directories() {
    log_info "Creating directories..."
    
    mkdir -p "$MCP_LOG_DIR"
    mkdir -p "$MCP_CONFIG_DIR"
    mkdir -p "$(dirname $MCP_PID_FILE)"
    
    # Set appropriate permissions
    chmod 750 "$MCP_LOG_DIR"
    chmod 750 "$MCP_CONFIG_DIR"
}

# Generate secure API key
generate_api_key() {
    # Generate a secure random API key
    API_KEY=$(openssl rand -hex 32)
    echo "$API_KEY" > "$MCP_CONFIG_DIR/api-key"
    chmod 600 "$MCP_CONFIG_DIR/api-key"
    log_info "Generated API key: $MCP_CONFIG_DIR/api-key"
}

# Create systemd service file
create_systemd_service() {
    log_info "Creating systemd service..."
    
    cat > /etc/systemd/system/redi-orm-mcp.service << EOF
[Unit]
Description=RediORM MCP Server
After=network.target postgresql.service mysql.service

[Service]
Type=simple
User=rediorm
Group=rediorm
WorkingDirectory=/opt/redi-orm
ExecStart=/usr/local/bin/redi-orm mcp \\
    --db="\${DATABASE_URI}" \\
    --schema=/opt/redi-orm/schema.prisma \\
    --transport=http \\
    --port=$MCP_PORT \\
    --enable-auth \\
    --api-key="\$(cat $MCP_CONFIG_DIR/api-key)" \\
    --read-only \\
    --rate-limit=100 \\
    --allowed-tables="\${ALLOWED_TABLES}" \\
    --allowed-hosts="\${ALLOWED_HOSTS}" \\
    --log-level=info

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$MCP_LOG_DIR

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Restart policy
Restart=always
RestartSec=5

# Environment
EnvironmentFile=$MCP_CONFIG_DIR/environment

# Logging
StandardOutput=append:$MCP_LOG_DIR/mcp.log
StandardError=append:$MCP_LOG_DIR/mcp-error.log

[Install]
WantedBy=multi-user.target
EOF

    # Create environment file
    cat > "$MCP_CONFIG_DIR/environment" << EOF
# RediORM MCP Environment Configuration
# Edit these values for your environment

# Database connection (use read-only user!)
DATABASE_URI=postgresql://readonly:password@localhost:5432/production

# Allowed tables (comma-separated)
ALLOWED_TABLES=users,products,orders,inventory

# Allowed hosts (comma-separated)
ALLOWED_HOSTS=api.example.com,assistant.example.com,localhost
EOF

    chmod 600 "$MCP_CONFIG_DIR/environment"
    log_info "Created systemd service: /etc/systemd/system/redi-orm-mcp.service"
}

# Create dedicated user
create_user() {
    if ! id "rediorm" &>/dev/null; then
        log_info "Creating rediorm user..."
        useradd -r -s /bin/false -d /opt/redi-orm rediorm
    fi
}

# Setup nginx reverse proxy
setup_nginx() {
    log_info "Setting up nginx configuration..."
    
    cat > /etc/nginx/sites-available/redi-orm-mcp << 'EOF'
# Rate limiting
limit_req_zone $binary_remote_addr zone=mcp_limit:10m rate=10r/s;

# Upstream MCP server
upstream mcp_backend {
    server 127.0.0.1:8443;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name mcp.example.com;
    
    # SSL configuration
    ssl_certificate /etc/ssl/certs/mcp.example.com.crt;
    ssl_certificate_key /etc/ssl/private/mcp.example.com.key;
    
    # Security headers
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Logging
    access_log /var/log/nginx/mcp-access.log;
    error_log /var/log/nginx/mcp-error.log;
    
    # API endpoint
    location / {
        # Rate limiting
        limit_req zone=mcp_limit burst=20 nodelay;
        
        # Only allow POST requests
        limit_except POST {
            deny all;
        }
        
        # Proxy settings
        proxy_pass http://mcp_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 10s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        
        # Buffer settings
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }
    
    # SSE endpoint
    location /events {
        # Rate limiting
        limit_req zone=mcp_limit burst=5 nodelay;
        
        # SSE specific settings
        proxy_pass http://mcp_backend/events;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Cache-Control "no-cache";
        proxy_set_header X-Accel-Buffering "no";
        
        # Long timeout for SSE
        proxy_read_timeout 3600s;
        
        # Disable buffering for SSE
        proxy_buffering off;
    }
    
    # Health check endpoint
    location /health {
        access_log off;
        return 200 "OK\n";
        add_header Content-Type text/plain;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name mcp.example.com;
    return 301 https://$server_name$request_uri;
}
EOF

    log_info "Created nginx configuration: /etc/nginx/sites-available/redi-orm-mcp"
    log_warn "Remember to:"
    log_warn "  1. Update server_name in nginx config"
    log_warn "  2. Install SSL certificates"
    log_warn "  3. Enable the site: ln -s /etc/nginx/sites-available/redi-orm-mcp /etc/nginx/sites-enabled/"
    log_warn "  4. Test nginx config: nginx -t"
    log_warn "  5. Reload nginx: systemctl reload nginx"
}

# Setup log rotation
setup_logrotate() {
    log_info "Setting up log rotation..."
    
    cat > /etc/logrotate.d/redi-orm-mcp << EOF
$MCP_LOG_DIR/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 rediorm rediorm
    sharedscripts
    postrotate
        systemctl reload redi-orm-mcp >/dev/null 2>&1 || true
    endscript
}
EOF

    log_info "Created logrotate configuration: /etc/logrotate.d/redi-orm-mcp"
}

# Setup monitoring
setup_monitoring() {
    log_info "Creating monitoring script..."
    
    cat > /usr/local/bin/check-mcp-health << 'EOF'
#!/bin/bash
# MCP Health Check Script

API_KEY=$(cat /etc/redi-orm/api-key)
ENDPOINT="http://localhost:8443"

# Check if service is running
if ! systemctl is-active --quiet redi-orm-mcp; then
    echo "CRITICAL: MCP service is not running"
    exit 2
fi

# Check API endpoint
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}' \
    "$ENDPOINT/")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" != "200" ]]; then
    echo "CRITICAL: API returned HTTP $HTTP_CODE"
    exit 2
fi

if echo "$BODY" | grep -q '"error"'; then
    echo "WARNING: API returned error: $BODY"
    exit 1
fi

echo "OK: MCP service is healthy"
exit 0
EOF

    chmod +x /usr/local/bin/check-mcp-health
    log_info "Created health check script: /usr/local/bin/check-mcp-health"
}

# Setup firewall rules
setup_firewall() {
    log_info "Setting up firewall rules..."
    
    # Allow HTTPS
    ufw allow 443/tcp comment 'MCP HTTPS'
    
    # Deny direct access to MCP port (only nginx should access it)
    ufw deny $MCP_PORT/tcp comment 'MCP Direct - Blocked'
    
    log_info "Firewall rules configured"
}

# Create backup script
create_backup_script() {
    log_info "Creating backup script..."
    
    cat > /usr/local/bin/backup-mcp-config << 'EOF'
#!/bin/bash
# MCP Configuration Backup Script

BACKUP_DIR="/var/backups/redi-orm"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/mcp-config-$TIMESTAMP.tar.gz"

mkdir -p "$BACKUP_DIR"

# Create backup
tar -czf "$BACKUP_FILE" \
    /etc/redi-orm/ \
    /etc/systemd/system/redi-orm-mcp.service \
    /etc/nginx/sites-available/redi-orm-mcp \
    /etc/logrotate.d/redi-orm-mcp \
    2>/dev/null

# Keep only last 30 backups
ls -t "$BACKUP_DIR"/mcp-config-*.tar.gz | tail -n +31 | xargs -r rm

echo "Backup created: $BACKUP_FILE"
EOF

    chmod +x /usr/local/bin/backup-mcp-config
    
    # Add to crontab
    echo "0 2 * * * /usr/local/bin/backup-mcp-config" >> /etc/crontab
    
    log_info "Created backup script with daily cron job"
}

# Main installation
main() {
    log_info "Starting RediORM MCP Production Setup..."
    
    check_root
    setup_directories
    create_user
    generate_api_key
    create_systemd_service
    setup_nginx
    setup_logrotate
    setup_monitoring
    setup_firewall
    create_backup_script
    
    # Enable service
    systemctl daemon-reload
    systemctl enable redi-orm-mcp
    
    log_info "Installation complete!"
    log_info ""
    log_info "Next steps:"
    log_info "1. Edit $MCP_CONFIG_DIR/environment with your database connection"
    log_info "2. Copy your schema.prisma to /opt/redi-orm/"
    log_info "3. Update nginx configuration with your domain"
    log_info "4. Install SSL certificates"
    log_info "5. Start the service: systemctl start redi-orm-mcp"
    log_info "6. Check logs: tail -f $MCP_LOG_DIR/mcp.log"
    log_info ""
    log_info "Security checklist:"
    log_info "✓ API key authentication enabled"
    log_info "✓ Read-only database access"
    log_info "✓ Rate limiting configured"
    log_info "✓ HTTPS only with nginx"
    log_info "✓ Systemd security hardening"
    log_info "✓ Log rotation configured"
    log_info "✓ Health monitoring available"
    log_info "✓ Firewall rules applied"
    log_info "✓ Daily backup scheduled"
}

# Run main installation
main "$@"