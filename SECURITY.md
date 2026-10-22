# Security Policy

## Supported Versions

Security updates and patches are provided for the current active release on the primary branch:

| Version | Supported          |
| ------- | ------------------ |
| 0.4.x   | :white_check_mark: |
| < 0.4.0 | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in BlockAds Desktop, please do **NOT** open a public issue.

Instead, please send a detailed report via email to:
**nuruddinrusydiilham@gmail.com**

Please include:
- A description of the vulnerability and potential impact.
- Affected components or functions.
- Step-by-step reproduction instructions or proof-of-concept.
- Any suggested mitigations.

We will acknowledge receipt within 48 hours and coordinate remediation before public disclosure.

## Security Architecture & Design Boundaries

BlockAds Desktop is engineered with defense-in-depth principles:

1. **DNS Privacy & Encryption**:
   - All external DNS queries are resolved exclusively over encrypted **DNS-over-HTTPS (DoH)**.
   - Strict TLS certificate verification using standard Go crypto and system root CAs.
   - Zero plaintext DNS fallback for user queries.

2. **Loopback Server Isolation**:
   - The local DNS proxy binds strictly to `127.0.0.1:5335` (UDP/TCP loopback), preventing exposure to external network interfaces.

3. **Filter Download & SSRF Protection**:
   - Filter-list downloading employs strict redirect limits, destination IP validation (blocking private/loopback/link-local addresses to prevent SSRF), bounded response reader limits (preventing decompression/memory bombs), and HTTP Content-Type inspection (rejecting HTML error pages).

4. **Input Sanitization & Safe Parsing**:
   - Filter metadata comments (`! Title:`, `! Description:`, etc.) are parsed as bounded, untrusted plain text. They are never executed, passed to shells, or rendered as unescaped HTML.

5. **Storage Security**:
   - All SQLite queries use parameterized prepared statements, preventing SQL injection.
   - Application data is isolated to per-user `%LOCALAPPDATA%\BlockAds Desktop` or the local `data/` folder in portable mode.

6. **Non-Privileged Execution**:
   - BlockAds Desktop operates entirely under standard non-administrator user privileges.
   - Startup persistence utilizes the current user's registry hive (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`), never modifying machine-wide keys or services.
