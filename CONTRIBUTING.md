# Contributing

Thanks for contributing! This project aims to stay simple and maintainable.
Please follow the guidelines below.

---

## 1. Issues

- Create an issue before starting significant work.
- Keep issues focused and small.
- Use clear titles:
  - `Bug: Reconcile fails when cpu is set to a negative value`
  - `Feature: Add optional metrics integration with prometheus`

---

## 2. Branch Naming

Use the issue number in your branch name:

<issue-number>-short-description

Example:

123-fix-login-validation

---

## 3. Pull Requests

Every PR must:

- Reference the issue using:

  Closes #<issue-number>

- Clearly explain what changed
- Be small and focused

Large PRs are harder to review and more likely to introduce bugs.

> [!TIP]
> There is a pull request template to help with this

---

## 4. Code Style

- Follow existing patterns in the project.
- Prefer readability over cleverness.
- Keep functions small and focused.
- Avoid unnecessary dependencies.

If you’re unsure, match surrounding code.

---

## 5. Testing

- Test your changes locally before opening a PR.
- Add tests when introducing new logic or fixing bugs.
- If skipping tests, explain why in the PR.

---

## 6. Reviews

- Be constructive and respectful.
- Focus on the code, not the person.
- If feedback is given, respond or apply changes promptly.

---

## 7. Merging

- Use **Squash and merge** unless there’s a reason not to.
- The PR title should clearly describe the change.
- Ensure CI passes before merging.

---

## 8. Keep It Simple

We value:
- Small changes
- Clear communication
- Clean history
- Maintainable code
