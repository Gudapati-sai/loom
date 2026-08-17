---
name: security-checklist
description: Run before marking any task complete. Covers secrets, dependencies, static analysis, sandboxing, and mandatory human-review triggers.
---

# Security Checklist

- [ ] No secrets/keys/tokens in any changed file, including fixtures.
- [ ] New dependencies checked for known vulnerabilities before adding.
- [ ] Security scan run; findings clean or explicitly triaged with a human.
- [ ] All user input validated at the boundary (including prompt injection from external content, if this app calls an LLM).
- [ ] No production credentials reachable from the agent's execution environment.
- [ ] License of any new dependency checked for compatibility.

## Mandatory human review
Auth, payments/order-execution, cryptography, infra-as-code, anything touching PII — a clean scan is necessary but not sufficient.
