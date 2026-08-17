---
name: documentation-driven-development
description: Use before starting any non-trivial feature or bug fix. Ensures a written spec exists and is the source of truth the implementation and tests are checked against.
---

# Documentation-Driven Development

## When to use
Before writing implementation code for a feature, a behavior-changing bug fix, or a public interface change.

## Steps
1. Draft the spec in `docs/prd/<date>-<name>.md` from the template. Problem, constraints, interface, numbered acceptance criteria.
2. Human sign-off before implementation starts.
3. Derive the test checklist from the acceptance criteria.
4. Implement against the spec, not your interpretation of the original request.
5. Review spec, tests, and code together — a mismatch is a defect.
6. Keep specs versioned in the repo, not in chat history.

## Anti-pattern to avoid
Writing code first and reverse-engineering a spec afterward.
