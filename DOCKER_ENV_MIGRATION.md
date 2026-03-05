# Docker Environment Variable Migration

## 🎯 Problem Solved

The previous Docker Compose configuration used **volume mounts** to inject configuration files and SSL certificates. This approach fails in restricted deployment environments (CI/CD pipelines, managed services, security-hardened platforms) that don't allow volume mounts.

**Error you were seeing:**
```
ERROR | Sanitizer check error volumes is not allowed in docker-compose.yml
```

## ✅ Solution Implemented

The application now supports **environment variable-based configuration**, which is:
- ✅ Compatible with restricted deployment environments
- ✅ More secure (no file system dependencies)
- ✅ Cloud-native and 12-factor app compliant
- ✅ Easier to manage in CI/CD pipelines

## 📝 What Changed

### 1. Docker Compose (`docker-compose.yml`)
- **Removed:** Volume mounts for `config.yaml` and SSL certificate
- **Removed:** Obsolete `version: '3.8'` attribute
- **Removed:** Unused named volumes section
- **Added:** Environment variables for all configuration

### 2. Dockerfile (`docker/Dockerfile`)
- **Added:** Entrypoint script that generates `config.yaml` from environment variables
- **Added:** Support for base64-encoded SSL certificate

### 3. New Files
- **`docker/entrypoint.sh`**: Generates `config.yaml` at runtime from env vars
- **`scripts/prepare-env.sh`**: Interactive helper to create `.env` file
- **`DOCKER_ENV_MIGRATION.md`**: This documentation

### 4. Updated Files
- **`.env.example`**: Now includes all necessary environment variables
- **`DEPLOYMENT.md`**: Added quick start guide for environment-based deployment

## 🚀 Quick Migration Guide

### If you were using volume-based configuration:

**Before:**
```yaml
volumes:
  - ./config.yaml:/app/config.yaml:ro
  - ${HOME}/.cloud-certs/root.crt:/app/certs/root.crt:ro
```

**After:**
```bash
# 1. Encode your SSL certificate
cat ~/.cloud-certs/root.crt | base64 | tr -d '\n' > cert.txt

# 2. Create .env file with environment variables
DATABASE_DSN=postgresql://user:pass@host:5432/db?sslmode=verify-full
PGSSLROOTCERT_BASE64=$(cat cert.txt)
MAILBOXES_JSON='[{"email":"user@domain.com","password":"pass","host":"mail.hosting.reg.ru","port":993}]'
POLL_INTERVAL=1m
TZ=Europe/Moscow
```

### Using the helper script (Recommended):

```bash
# Interactive setup
./scripts/prepare-env.sh

# Follow the prompts to configure:
# - Database connection
# - SSL certificate
# - Email mailboxes
# - Application settings
```

## 📋 Environment Variables Reference

### Database Configuration
```bash
DATABASE_DSN=postgresql://user:pass@host:5432/db?sslmode=verify-full
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=5m
```

### SSL Certificate (Base64 Encoded)
```bash
# Encode your certificate
PGSSLROOTCERT_BASE64=LS0tLS1CRUdJTi...

# How to encode:
cat /path/to/root.crt | base64 | tr -d '\n'
```

### Email Mailboxes (JSON Array)
```bash
# Single mailbox
MAILBOXES_JSON='[{"email":"user@domain.com","password":"pass","host":"mail.hosting.reg.ru","port":993}]'

# Multiple mailboxes
MAILBOXES_JSON='[{"email":"user1@domain.com","password":"pass1","host":"mail.hosting.reg.ru","port":993},{"email":"user2@domain.com","password":"pass2","host":"mail.hosting.reg.ru","port":993}]'
```

### Application Settings
```bash
POLL_INTERVAL=1m           # How often to check for emails
TZ=Europe/Moscow           # Timezone
```

## 🔧 How It Works

1. **Build time:** Dockerfile copies the `entrypoint.sh` script into the image
2. **Runtime:** When container starts, `entrypoint.sh` runs first
3. **Configuration generation:**
   - Reads environment variables
   - Decodes base64-encoded SSL certificate
   - Parses JSON mailboxes array
   - Generates `/app/config.yaml`
4. **Application start:** Your Go app reads the generated config file

## 🎨 Advantages of This Approach

### For Development
- Easy to switch between different configurations
- No need to manage multiple config files
- Works seamlessly with `.env` files

### For Production
- Compatible with Docker secrets and secrets management systems
- No sensitive files in the repository
- Environment-specific configuration without code changes

### For CI/CD
- No volume mount restrictions
- Easy to inject secrets from CI/CD variables
- Platform-agnostic deployment

## 🔄 Backward Compatibility

The old volume-based approach still works if you:
1. Remove the environment variables from `docker-compose.yml`
2. Re-add the volume mounts
3. Provide `config.yaml` and SSL certificate as files

However, **this is not recommended** for restricted environments.

## 📚 Additional Resources

- See [DEPLOYMENT.md](DEPLOYMENT.md) for full deployment guide
- See [.env.example](.env.example) for all environment variables
- See [config.example.yaml](config.example.yaml) for config file format

## 🆘 Troubleshooting

### Issue: Container fails to start

**Check logs:**
```bash
docker compose logs app
```

**Common causes:**
- Missing required environment variables
- Invalid JSON in `MAILBOXES_JSON`
- Invalid base64 in `PGSSLROOTCERT_BASE64`
- Invalid database DSN

### Issue: Can't connect to PostgreSQL

**Check certificate:**
```bash
# Verify base64 encoding
echo $PGSSLROOTCERT_BASE64 | base64 -d | head -n 1
# Should show: -----BEGIN CERTIFICATE-----
```

**Check from container:**
```bash
docker compose exec app sh
ls -la /app/certs/root.crt
cat /app/certs/root.crt | head -n 1
```

### Issue: Mailboxes not configured

**Validate JSON:**
```bash
echo $MAILBOXES_JSON | python3 -m json.tool
```

Should output formatted JSON without errors.

### Issue: Config not generated

**Check entrypoint script:**
```bash
docker compose exec app sh
cat /app/config.yaml
```

If the file is missing or empty, check environment variables:
```bash
docker compose exec app env | grep -E 'DATABASE|MAILBOX|POLL'
```

## 🔐 Security Best Practices

1. **Never commit `.env` to git**
   - Already in `.gitignore`
   - Use `.env.example` as a template

2. **Use secrets management in production**
   - Docker Secrets
   - Kubernetes Secrets
   - Cloud provider secrets (AWS Secrets Manager, GCP Secret Manager, etc.)

3. **Rotate credentials regularly**
   - Update `.env` file
   - Restart container: `docker compose restart app`

4. **Limit environment variable exposure**
   - Use `docker compose config` carefully (it exposes all variables)
   - Don't log environment variables

## 📞 Support

If you encounter issues:
1. Check this guide
2. Review [DEPLOYMENT.md](DEPLOYMENT.md)
3. Check container logs: `docker compose logs -f app`
4. Create an issue with logs and configuration (redact secrets!)
