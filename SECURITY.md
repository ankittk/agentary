# Security Policy

## Supported versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| older   | :x:                |

We recommend always running the latest release. Pre-release and development branches are not supported for production use.

## Reporting a vulnerability

We take security seriously. If you believe you have found a security vulnerability, please report it responsibly.

### How to report

- **Do not** open a public GitHub issue for security vulnerabilities.
- **Do** report privately by:
  - Opening a **private security advisory** on GitHub: go to the repository → **Security** → **Advisories** → **Report a vulnerability**, or
  - Emailing the maintainers (contact details may be listed in the repository profile or CODE_OF_CONDUCT).

### What to include

- Description of the vulnerability and how it can be reproduced.
- Impact (e.g. privilege escalation, data exposure, denial of service).
- Steps or proof-of-concept if available.
- Your name/handle for acknowledgment, if you want to be credited.

### What to expect

- We will acknowledge your report as soon as possible.
- We will work with you to understand and validate the issue.
- We will provide updates on our progress and any fix or advisory.
- We will credit you in the advisory/release notes unless you prefer to remain anonymous.

### Disclosure

We aim to fix critical issues promptly and then disclose them in a coordinated way (e.g. with a release and security advisory). We may delay disclosure briefly if a fix is in progress and disclosure would put users at risk.

## Security-related design

- **Sandboxing**: Agent subprocesses run in a restricted environment. See [docs/content/sandboxing.md](docs/content/sandboxing.md) for current behavior and limitations.
- **API authentication**: Optional `AGENTARY_API_KEY`; when set, API requests require `X-API-Key` or `?api_key=`. `/health` and `/metrics` remain unauthenticated.
- **Data**: Database and sensitive config live under `AGENTARY_HOME` (e.g. `~/.agentary/protected/`). Keep this directory and backups restricted.

Improvements to sandboxing and defense-in-depth are planned (see the project roadmap).
