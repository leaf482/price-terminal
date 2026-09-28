# Development Workflow

## Required review and commit cycle

Follow this sequence for every task:

1. **ChatGPT reviews the current repository and state.** Review relevant files, prior results, open issues, and working-tree changes rather than assuming the plan has already been implemented.
2. **ChatGPT selects one small next task.** Use [TASKS.md](TASKS.md), current dependencies, and findings from the last review. State the requested scope and acceptance criteria.
3. **The user sends a concise task prompt to Codex IDE.** Include the task ID, scope, constraints, and acceptance criteria.
4. **Codex reads existing code before changing anything.** Inspect applicable repository guidance, planning documents, relevant implementation, tests, and git status. Preserve unrelated user changes. For an empty repository or documentation task, inspect the existing files instead.
5. **Codex implements only the requested scope.** Make the smallest coherent change that meets the acceptance criteria. Do not automatically proceed to later roadmap tasks.
6. **Codex runs relevant tests and checks.** Choose checks appropriate to the change; report failures and unavailable checks honestly. Documentation-only tasks need consistency, scope, and diff checks, not application setup.
7. **Codex reports changed files, design choices, verification results, and git diff/status.** Include a concise diff summary, `git diff --stat`, and `git status`. List untracked files explicitly because normal `git diff` does not include them. Provide sufficient diff/code evidence for review and identify remaining uncertainties.
8. **The user sends the result back to ChatGPT.** Include the report and relevant diff/code/test output, including newly created files.
9. **ChatGPT reviews the diff, code, and tests.** Check scope, acceptance criteria, domain semantics, regressions, and whether the reported verification supports the result.
10. **Fixes go back to Codex if needed.** Keep fixes scoped to review findings, rerun relevant checks, and repeat the report and review cycle.
11. **Only after review passes does the user commit manually.** ChatGPT review approval precedes the user's commit.
12. **Codex must never create the git commit.** This applies to initial work, fixes, and later roadmap tasks. Do not auto-commit on completion or review approval.

## Task sizing and acceptance criteria

One task should normally map to one small commit. Each task needs concrete acceptance criteria before implementation begins. If a task becomes too large, propose a smaller slice and update the roadmap through review instead of delivering multiple unrequested features.

The roadmap criteria describe outcomes, not exhaustive implementation instructions. Refine them with the current repository state when preparing each task prompt. Select necessary tools and details at that point rather than locking every future task to today's assumptions.

Example task prompt structure:

```text
Task <ID>: <short title>
Read existing code, relevant docs, and git status first.
Scope: <one bounded change>
Acceptance criteria: <observable outcomes and relevant verification>
Constraints: no unrelated changes; no git commit.
Report changed files, decisions, checks, git diff --stat, and git status.
```

## Engineering priorities

- Correctness and data trust come before feature count.
- Preserve the domain distinctions in [DOMAIN_MODEL.md](DOMAIN_MODEL.md), particularly immutable observations, price meanings, money, currency, conditions, and unknown states.
- Avoid over-engineering. Do not introduce infrastructure, dependencies, or abstractions for hypothetical scale.
- Minimize unrelated changes. Avoid incidental formatting, renaming, dependency upgrades, or cleanup outside the requested scope.
- Test behavior and failure modes relevant to the change. Prefer deterministic provider fixtures over live-source dependencies in routine tests.
- Keep credentials and private data out of repository files and reports.
- Update relevant planning documents when an accepted implementation decision changes the source of truth.

## Completion report

Each Codex result should include:

1. Files created or changed, with a short purpose for each.
2. Design choices, assumptions, and any departures from the task prompt.
3. Verification commands and outcomes, including failures or checks not run.
4. `git diff --stat` and `git status`, plus explicit new-file coverage when files are untracked.
5. Outstanding review items, if any, and confirmation that no commit was created.

Task completion means the scoped work is ready for review, not that it has been committed or that the next task is authorized.
