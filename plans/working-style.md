# Working style

## Git
- Commits: 2-5 words, lowercase, imperative ("fix engine index panics").
- New feature → new worktree: `wt switch -c <branch>`.
- Dependent changes → stacked PRs (`gh stack`) when it helps review.

## PRs
- Open with `gh pr create`; no need to ask first.
- Title: 3-6 words, lowercase.
- Description: 1-3 sentences.

## Dev
- Prefer simple over complex. Smallest change that solves the problem.
- Keep things short and sweet: code, comments, docs.
