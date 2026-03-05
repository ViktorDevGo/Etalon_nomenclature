#!/bin/bash
# Helper script to prepare environment variables for Docker deployment

set -e

echo "================================================"
echo "  Etalon Nomenclature - Environment Setup"
echo "================================================"
echo ""

# Check if .env already exists
if [ -f .env ]; then
    read -p ".env file already exists. Overwrite? (y/N): " overwrite
    if [ "$overwrite" != "y" ] && [ "$overwrite" != "Y" ]; then
        echo "Aborted."
        exit 0
    fi
fi

# Create .env file
cat > .env <<'EOF'
# ================================
# DATABASE CONFIGURATION
# ================================
EOF

# Get database DSN
echo ""
echo "Step 1: Database Configuration"
echo "--------------------------------"
read -p "Enter PostgreSQL DSN: " db_dsn
echo "DATABASE_DSN=$db_dsn" >> .env
echo "DATABASE_MAX_OPEN_CONNS=25" >> .env
echo "DATABASE_MAX_IDLE_CONNS=5" >> .env
echo "DATABASE_CONN_MAX_LIFETIME=5m" >> .env
echo "" >> .env

# Get SSL certificate
echo ""
echo "Step 2: SSL Certificate"
echo "--------------------------------"
read -p "Enter path to SSL certificate file: " cert_path

if [ -f "$cert_path" ]; then
    cert_encoded=$(cat "$cert_path" | base64 | tr -d '\n')
    cat >> .env <<EOF
# ================================
# SSL CERTIFICATE (Base64 Encoded)
# ================================
PGSSLROOTCERT_BASE64=$cert_encoded

EOF
    echo "✓ Certificate encoded successfully"
else
    echo "✗ Certificate file not found: $cert_path"
    cat >> .env <<EOF
# ================================
# SSL CERTIFICATE (Base64 Encoded)
# ================================
PGSSLROOTCERT_BASE64=

EOF
fi

# Get mailboxes
echo ""
echo "Step 3: Email Mailboxes"
echo "--------------------------------"
read -p "How many mailboxes do you want to configure? " mailbox_count

mailboxes_json="["
for ((i=1; i<=mailbox_count; i++)); do
    echo ""
    echo "Mailbox #$i:"
    read -p "  Email: " email
    read -sp "  Password: " password
    echo ""
    read -p "  IMAP Host [mail.hosting.reg.ru]: " host
    host=${host:-mail.hosting.reg.ru}
    read -p "  IMAP Port [993]: " port
    port=${port:-993}

    if [ $i -gt 1 ]; then
        mailboxes_json+=","
    fi
    mailboxes_json+="{\"email\":\"$email\",\"password\":\"$password\",\"host\":\"$host\",\"port\":$port}"
done
mailboxes_json+="]"

cat >> .env <<EOF
# ================================
# EMAIL MAILBOXES (JSON Format)
# ================================
MAILBOXES_JSON='$mailboxes_json'

EOF

# Application settings
echo ""
echo "Step 4: Application Settings"
echo "--------------------------------"
read -p "Poll interval (e.g., 1m, 5m, 30s) [1m]: " poll_interval
poll_interval=${poll_interval:-1m}
read -p "Timezone [Europe/Moscow]: " timezone
timezone=${timezone:-Europe/Moscow}

cat >> .env <<EOF
# ================================
# APPLICATION SETTINGS
# ================================
POLL_INTERVAL=$poll_interval
TZ=$timezone
EOF

echo ""
echo "================================================"
echo "✓ Configuration saved to .env"
echo "================================================"
echo ""
echo "Next steps:"
echo "  1. Review the .env file: nano .env"
echo "  2. Build the Docker image: docker compose build"
echo "  3. Start the service: docker compose up -d"
echo "  4. Check logs: docker compose logs -f app"
echo ""
