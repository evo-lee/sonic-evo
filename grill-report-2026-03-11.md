---
plugin: grill
version: 1.2.0
date: 2026-03-11
target: /Users/l/Documents/code/sonic
style: Architecture Review + Rewrite Plan
addons: [Scale stress, Hidden costs, Principle violations, Strangler fig, Success metrics, Before vs after, Assumptions audit, Compact & optimize]
agents: [architecture, security, testing, error-handling]
---

# Sonic Architecture Review & Rewrite Plan

**Date**: 2026-03-11  
**Target**: /Users/l/Documents/code/sonic  
**Review Style**: Architecture Review + Rewrite Plan  
**Agents**: architecture, security, testing, error-handling

---

## Executive Summary

### Overall Verdict

The Sonic blogging platform demonstrates a **solid architectural foundation** with clean layered separation (Handler → Service → DAL → Model), modern Go practices (Uber FX dependency injection, GORM code generation, structured logging), and good security fundamentals (bcrypt password hashing, zip slip protection). However, **critical gaps in testing (2.2% coverage), security vulnerabilities (path traversal, plaintext category passwords), and architectural violations (global mutable state, god objects) significantly undermine production readiness**.

**Biggest Risk**: The combination of zero service layer test coverage (34 services, 0 tests) and critical security vulnerabilities (path traversal in backup downloads, no CSRF protection) creates a high-risk deployment scenario where business logic failures and security exploits cannot be detected before production.

### Top 3 Actions

1. **Fix Critical Security Vulnerabilities (1-2 days, HIGH confidence)**
   - Path traversal in backup file downloads (CRITICAL)
   - Hash category passwords with bcrypt (CRITICAL)
   - Implement CSRF protection (HIGH)
   
   **Why**: These are exploitable vulnerabilities with clear attack vectors. Path traversal allows arbitrary file reads, plaintext passwords expose protected content, and missing CSRF enables unauthorized actions.

2. **Establish Testing Foundation (2-3 weeks, MEDIUM confidence)**
   - Add service layer tests for authentication and post management
   - Enable coverage reporting in CI with 20% minimum threshold
   - Create database integration tests with testcontainers
   
   **Why**: Zero test coverage for business logic means every deployment is a gamble. Starting with authentication and post management covers the highest-risk code paths.

3. **Remove Global Mutable State (2-3 days, HIGH confidence)**
   - Eliminate fx.Populate for dal.DB and eventBus
   - Pass dependencies explicitly through constructors
   - Fix handler layer direct DAL access
   
   **Why**: Global state breaks dependency injection, makes testing impossible, and introduces race conditions. This is a foundational issue that blocks other improvements.

### Confidence Levels

- **Security fixes**: HIGH - Clear vulnerabilities with known solutions
- **Testing foundation**: MEDIUM - Requires architectural decisions on test strategy (mocking vs integration)
- **Global state removal**: HIGH - Well-understood refactoring with clear benefits
- **Service decomposition**: LOW - Requires domain expertise to split correctly (would increase with stakeholder input)
- **Performance improvements**: LOW - No load testing data to validate assumptions (would increase with production metrics)

### Paranoid Verdict

**The single scariest thing**: Path traversal in backup file download (`/service/impl/backup.go:119-128`) combined with admin authentication bypass potential. An attacker who gains admin access (via session hijacking, CSRF, or credential stuffing due to no rate limiting) can read arbitrary files from the filesystem including:
- `/etc/passwd` and `/etc/shadow` (system credentials)
- Database files (`sonic.db` with all user data)
- Configuration files with secrets (`conf/config.yaml`)
- Application source code (intellectual property theft)

This is a **complete system compromise** scenario with data exfiltration, credential theft, and potential lateral movement to other systems.

---

