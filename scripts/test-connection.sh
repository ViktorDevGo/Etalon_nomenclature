#!/bin/bash

# Script to test connections before deployment

set -e

echo "=== Testing Connections ==="
echo ""

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo "❌ config.yaml not found"
    exit 1
fi
echo "✅ config.yaml found"

# Check if SSL certificate exists
if [ ! -f "$HOME/.cloud-certs/root.crt" ]; then
    echo "❌ SSL certificate not found at $HOME/.cloud-certs/root.crt"
    exit 1
fi
echo "✅ SSL certificate found"

# Test IMAP connection
echo ""
echo "Testing IMAP connection to mail.hosting.reg.ru:993..."
if command -v nc &> /dev/null; then
    if nc -zv mail.hosting.reg.ru 993 2>&1 | grep -q "succeeded"; then
        echo "✅ IMAP port 993 is accessible"
    else
        echo "⚠️  IMAP port 993 might not be accessible"
    fi
else
    echo "⚠️  netcat not installed, skipping IMAP test"
fi

# Test PostgreSQL connection
echo ""
echo "Testing PostgreSQL connection..."
if command -v psql &> /dev/null; then
    # Extract DSN from config.yaml
    DSN=$(grep -A 1 "database:" config.yaml | grep "dsn:" | sed 's/.*dsn: "\(.*\)"/\1/')

    if [ -n "$DSN" ]; then
        export PGSSLROOTCERT="$HOME/.cloud-certs/root.crt"
        if psql "$DSN" -c "SELECT 1;" &> /dev/null; then
            echo "✅ PostgreSQL connection successful"
        else
            echo "❌ PostgreSQL connection failed"
            exit 1
        fi
    else
        echo "⚠️  Could not extract DSN from config.yaml"
    fi
else
    echo "⚠️  psql not installed, skipping PostgreSQL test"
fi

echo ""
echo "=== All tests completed ==="
