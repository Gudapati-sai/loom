---
name: anti-hallucination
description: Standing discipline for every task. Grounds claims in verifiable sources, verifies by execution instead of narration, and requires explicit uncertainty over confident guessing.
---

# Anti-Hallucination Protocol

## Rules
1. Ground, don't recall — read real files/docs instead of trusting memory.
2. Verify by execution — run it, report the actual result.
3. Small diffs — one logical change per turn.
4. Cite claims about the codebase — real grep/search output.
5. State uncertainty plainly — a guess presented as fact is the failure mode this exists to prevent.
6. Curate context — the specific files/output relevant to the task, not "the whole repo from memory."
7. Least privilege by default — read-only until a task explicitly needs more.

## Self-check before reporting complete
Did I read the actual current version of every file I claim to have changed? Did I run every verification I'm claiming passed? Is there anything I asserted that I didn't actually check?
