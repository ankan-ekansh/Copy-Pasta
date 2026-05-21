# Contributing to Copy-Pasta

## Branch Strategy

**All changes go through Pull Requests. No direct commits to `main`.**

### Workflow

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/short-description
   ```

2. **Make commits** on the feature branch.

3. **Push and open a PR**:
   ```bash
   git push -u origin feature/short-description
   gh pr create --title "feat: description" --body "What and why"
   ```

4. **CI runs automatically** — tests must pass before merge.

5. **Review required** — at least one approval before merging.

6. **Merge via squash** (keeps main history clean):
   ```bash
   gh pr merge --squash
   ```

### Branch Naming

| Prefix | Use |
|--------|-----|
| `feature/` | New functionality |
| `fix/` | Bug fixes |
| `refactor/` | Code improvements (no behavior change) |
| `docs/` | Documentation only |
| `infra/` | CI/CD, deployment, tooling |

### AI Agent Rules

AI assistants (Copilot, etc.) **must**:
- Always work on a feature branch, never commit directly to `main`
- Open a PR with a clear title and description
- Wait for CI to pass before requesting merge
- Include `Co-authored-by` trailer in commits

### Commit Messages

Use conventional commits:
```
feat: add clipboard paste support
fix: resolve WhatsApp line wrapping
refactor: extract braille pipeline into separate module
docs: update deployment guide
```
