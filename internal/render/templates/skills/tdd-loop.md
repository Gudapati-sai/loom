---
name: tdd-loop
description: Use whenever implementing new functionality or fixing a bug. Red-green-refactor loop with a mandatory human checkpoint on test correctness before implementation.
---

# TDD Loop

## The loop
1. Red — write tests expressing the acceptance criteria; they should fail for the right reason.
2. Checkpoint — a human confirms the tests test the right thing. Highest-leverage step in the loop.
3. Green — implement the minimum code to pass.
4. Verify for real — run the full suite + lint + type-check, paste actual output.
5. Refactor with the passing suite as a safety net, then re-run it.
6. Regression lock — a bug fix's first test must reproduce the bug and fail against old code.

## Anti-pattern to avoid
Writing tests and implementation in the same pass and presenting both as done together.
