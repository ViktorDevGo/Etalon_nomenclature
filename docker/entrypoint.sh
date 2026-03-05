#!/bin/sh
set -e

# Generate config.yaml from environment variables
cat > /app/config.yaml <<EOF
# Auto-generated configuration from environment variables
poll_interval: ${POLL_INTERVAL:-1m}

database:
  dsn: "${DATABASE_DSN}"
  max_open_conns: ${DATABASE_MAX_OPEN_CONNS:-25}
  max_idle_conns: ${DATABASE_MAX_IDLE_CONNS:-5}
  conn_max_lifetime: ${DATABASE_CONN_MAX_LIFETIME:-5m}
EOF

# Handle SSL certificate if provided (base64 encoded)
if [ -n "$PGSSLROOTCERT_BASE64" ]; then
  mkdir -p /app/certs
  echo "$PGSSLROOTCERT_BASE64" | base64 -d > /app/certs/root.crt
  echo "  ssl_root_cert: \"/app/certs/root.crt\"" >> /app/config.yaml
fi

# Add mailboxes from JSON if provided
if [ -n "$MAILBOXES_JSON" ]; then
  echo "" >> /app/config.yaml
  echo "mailboxes:" >> /app/config.yaml

  # Parse JSON array and convert to YAML
  # This uses a simple approach - for production, consider using yq or jq
  echo "$MAILBOXES_JSON" | sed 's/^\[//;s/\]$//' | tr '}' '\n' | while IFS= read -r mailbox; do
    if [ -n "$mailbox" ]; then
      email=$(echo "$mailbox" | sed -n 's/.*"email":"\([^"]*\)".*/\1/p')
      password=$(echo "$mailbox" | sed -n 's/.*"password":"\([^"]*\)".*/\1/p')
      host=$(echo "$mailbox" | sed -n 's/.*"host":"\([^"]*\)".*/\1/p')
      port=$(echo "$mailbox" | sed -n 's/.*"port":\([0-9]*\).*/\1/p')

      if [ -n "$email" ]; then
        cat >> /app/config.yaml <<MAILBOX
  - email: "$email"
    password: "$password"
    host: "${host:-mail.hosting.reg.ru}"
    port: ${port:-993}
MAILBOX
      fi
    fi
  done
fi

# Set config path
export CONFIG_PATH=/app/config.yaml

# Execute the application
exec "$@"
