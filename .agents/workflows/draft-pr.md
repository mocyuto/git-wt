---
description: Create o Update a draft pull request targeting the main branch
---

1. Stage and commit any uncommitted changes.
2. Push the current branch to the remote repository.
3. Analyze all diffs between the current branch and `main` to understand the full scope of changes.
4. Check if a pull request already exists for the current branch.
5. If a PR exists, compare its current title/body with the latest changes and update them if there are improvements or new details to add.
6. If no PR exists, create a new draft pull request targeting the `main` branch in **English**.
   - **Title**: Concise summary of the entire PR's changes.
   - **Body**: A detailed overview outlining the PR's purpose, key changes, and affected scope.
   - **Status**: Draft (when creating new).

Command to run:

- `git add . && (git diff-index --quiet HEAD || git commit -m "update")`
- `git push -u origin $(git branch --show-current)`
- `git diff main...$(git branch --show-current)` (to review all changes in the PR)
- If PR exists: `gh pr edit --title "<GENERATED_TITLE>" --body "<GENERATED_BODY>"`
- If PR does not exist: `gh pr create --draft --base main --title "<GENERATED_TITLE>" --body "<GENERATED_BODY>"`

_(Note: The AI agent should replace `<GENERATED_TITLE>` and `<GENERATED_BODY>` with appropriately generated Japanese text based on a thorough analysis of the total diff.)_
