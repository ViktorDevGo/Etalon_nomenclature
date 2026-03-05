# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-03-05

### Added
- Initial release
- IMAP email fetching from REG.RU mail server
- Excel (.xlsx) file parsing with streaming support
- PostgreSQL integration with SSL support
- Multiple mailbox support
- Message-ID based duplicate prevention
- Graceful shutdown handling
- Panic recovery mechanism
- Structured logging with zap
- Docker and Docker Compose support
- Automatic retry logic for IMAP connections
- Transaction-safe database operations
- Connection pooling for PostgreSQL
- Health checks in Docker Compose
- Resource limits configuration
- Comprehensive documentation
- Migration scripts
- Helper scripts for deployment
- Unit tests for core components

### Features
- **Email Processing**
  - Checks mailboxes every 1 minute (configurable)
  - Searches for emails from current day
  - Filters for .xlsx attachments
  - 10 MB file size limit
  - Automatic retry on IMAP failures (3 attempts)

- **Excel Parsing**
  - Processes all sheets in workbook
  - Streaming parser for memory efficiency
  - Handles optional columns (e.g., "Тип")
  - Supports comma-separated numbers
  - Automatic header row detection

- **Database**
  - Transactional inserts
  - Message-ID tracking
  - SSL/TLS support
  - Connection pooling
  - Automatic reconnection

- **Production Features**
  - Graceful shutdown
  - Context cancellation
  - Panic recovery with stack traces
  - Structured JSON logging
  - Health checks
  - Resource limits
  - Non-root Docker container
  - Multi-stage Docker build

### Security
- SSL/TLS for PostgreSQL connections
- Non-root user in Docker container
- Prepared statements for SQL queries
- File size validation
- No sensitive data in logs

### Documentation
- README.md — Main documentation
- QUICKSTART.md — Quick start guide
- DEPLOYMENT.md — Production deployment guide
- STRUCTURE.md — Project architecture
- CHANGELOG.md — This file
- Inline code comments

## [Unreleased]

### Planned
- [ ] Prometheus metrics export
- [ ] Configurable email search criteria
- [ ] Email body parsing
- [ ] Support for .xls files
- [ ] Webhook notifications
- [ ] Admin API
- [ ] Dashboard UI
- [ ] Multiple database support
- [ ] Email templates
- [ ] Attachment archive storage

---

## Version History

- **1.0.0** (2026-03-05) — Initial release
