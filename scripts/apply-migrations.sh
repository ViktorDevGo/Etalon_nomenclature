#!/bin/bash

# Script to apply database migrations

set -e

echo "=== Applying Database Migrations ==="
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "❌ psql is not installed"
    echo "Install it with: sudo apt install postgresql-client"
    exit 1
fi

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo "❌ config.yaml not found"
    exit 1
fi

# Check if SSL certificate exists
if [ ! -f "$HOME/.cloud-certs/root.crt" ]; then
    echo "❌ SSL certificate not found at $HOME/.cloud-certs/root.crt"
    exit 1
fi

# Extract DSN from config.yaml
DSN=$(grep -A 1 "database:" config.yaml | grep "dsn:" | sed 's/.*dsn: "\(.*\)"/\1/')

if [ -z "$DSN" ]; then
    echo "❌ Could not extract DSN from config.yaml"
    exit 1
fi

export PGSSLROOTCERT="$HOME/.cloud-certs/root.crt"

echo "Applying migrations..."
echo ""

# Apply each migration file
for migration in migrations/*.sql; do
    if [ -f "$migration" ]; then
        echo "Applying: $migration"
        psql "$DSN" -f "$migration"
        echo "✅ Applied: $migration"
        echo ""
    fi
done

echo "=== Migrations completed ==="
echo ""
echo "Listing tables:"
psql "$DSN" -c "\dt"
