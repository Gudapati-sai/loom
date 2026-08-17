---
name: prompt-loops
description: Reference for which reasoning loop to use for a given task shape — decomposition, ReAct, plan-execute-reflect, self-refine, reflexion, chain-of-verification, self-consistency, tree-of-thought.
---

# Prompting Techniques & Loops

| Loop | Use it when |
|---|---|
| Decomposition-first | Almost always, as the first move on anything non-trivial |
| ReAct (reason → act → observe) | Task depends on info you don't have yet |
| Plan → Execute → Reflect | Multi-step task spanning several files/systems |
| Self-Refine | Output where quality matters more than speed |
| Reflexion | Retrying a failed attempt |
| Chain-of-Verification | High-stakes factual claims |
| Self-Consistency | Ambiguous logic, tricky edge cases |
| Tree-of-Thought | Genuinely ambiguous design decisions with real trade-offs |

## Anti-pattern to avoid
Using a single confident pass on a high-stakes or ambiguous task just because it's faster.
