#!/bin/sh
set -e

# Build DATABASE_DSN from components if separate variables provided
if [ -n "$DB_HOST" ] && [ -n "$DB_USER" ] && [ -n "$DB_PASSWORD" ] && [ -n "$DB_NAME" ]; then
  # URL-encode password to handle special characters
  ENCODED_PASSWORD=$(printf '%s' "$DB_PASSWORD" | sed 's/</%3C/g; s/>/%3E/g; s/;/%3B/g; s/:/%3A/g; s/@/%40/g; s|/|%2F|g; s/?/%3F/g; s/#/%23/g; s/&/%26/g; s/=/%3D/g; s/ /%20/g')
  DATABASE_DSN="postgresql://${DB_USER}:${ENCODED_PASSWORD}@${DB_HOST}:${DB_PORT:-5432}/${DB_NAME}?sslmode=${DB_SSLMODE:-require}"
fi

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

# Add allowed senders if provided
if [ -n "$ALLOWED_SENDERS" ]; then
  echo "" >> /app/config.yaml
  echo "allowed_senders:" >> /app/config.yaml

  # Parse comma-separated list
  echo "$ALLOWED_SENDERS" | tr ',' '\n' | while IFS= read -r sender; do
    sender=$(echo "$sender" | xargs) # trim whitespace
    if [ -n "$sender" ]; then
      echo "  - \"$sender\"" >> /app/config.yaml
    fi
  done
fi

# Set config path
export CONFIG_PATH=/app/config.yaml

# Execute the application
exec "$@"
