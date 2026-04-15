# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in cyoda-go-spi, please report it privately.

**Email:** infosec@cyoda.com

**Please include:**

- A description of the vulnerability and its potential impact.
- Steps to reproduce or a minimal plugin that demonstrates the issue.
- The affected version(s).
- Any suggested mitigation.

**Please do not open public GitHub issues for security reports.** GitHub's [private security advisories](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) are also an acceptable channel.

## Response Expectations

- **Acknowledgement:** within 3 business days.
- **Initial assessment:** within 10 business days.
- **Coordinated disclosure:** we will work with you on a timeline. Typical window is 90 days from initial report to public disclosure.

## Supported Versions

cyoda-go-spi is pre-1.0. Security fixes are applied to the latest minor release. Older versions are not maintained.

## Scope

In scope:

- Vulnerabilities in the SPI contract itself (interfaces, value types, helpers) that could be exploited through a compliant plugin implementation.
- Supply-chain issues in direct dependencies (the SPI keeps dependencies minimal — primarily stdlib plus `google/uuid`).

Out of scope (report to the plugin's maintainer, or upstream):

- Vulnerabilities in specific plugin implementations — report to the plugin repo instead.
- Vulnerabilities in Go itself or in `google/uuid`.
