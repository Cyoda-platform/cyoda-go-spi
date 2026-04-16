---
paths:
  - "**/*.go"
---
# Logging Policy

Use `log/slog` exclusively. Never use `log.Printf` or `fmt.Printf` for operational logging.

## Log Levels

- **ERROR:** Something failed that shouldn't have. Requires investigation.
- **WARN:** Unexpected but recoverable. Might indicate a problem if repeated.
- **INFO:** High-level flow milestones. Reading INFO tells you what the system is doing.
- **DEBUG:** Detailed flow tracing with payload previews (truncate to ~200 chars).

## Rules

- Never log credentials, tokens, secrets, or signing keys at any level.
- Structured context: include `pkg`, plus identifying fields (entity/tenant/transaction IDs) as appropriate.
- One event = one log line at one level. Don't log the same event at INFO and DEBUG.
