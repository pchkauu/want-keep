# Security Policy

Treat security reports as urgent.

## Reporting a vulnerability

**Do not publish security vulnerabilities through public GitHub Issues.**

Use GitHub Private Vulnerability Reporting:

[Report a vulnerability](https://github.com/pchkauu/want-keep/security/advisories/new)

If that form is unavailable, contact the repository owner through GitHub at [@pchkauu](https://github.com/pchkauu). Do not include secrets or personal data in the first message.

Please include:

- a description of the issue
- affected files, endpoints, or docs, if known
- steps to reproduce without real credentials
- impact and any suggested fix

We will acknowledge the report, investigate, and coordinate disclosure after a fix is available.

## Owner setup

This repository does not yet have a dedicated security email.

**TODO for the repository owner:** enable [GitHub Private Vulnerability Reporting](https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/configuring-private-vulnerability-reporting-for-a-repository) for `pchkauu/want-keep`. Add a security contact email here only after that inbox exists.

## Handling secrets

Never commit real API keys, credentials, access tokens, cookies, or personal data.

If you find a secret in the repository, report it privately and rotate the credential. Do not open a public issue that repeats the secret.

Use synthetic fixtures. Remove secrets from logs and screenshots before sharing them.
