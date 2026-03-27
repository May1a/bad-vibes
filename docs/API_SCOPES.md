# API Scopes and Permissions

bad-vibes requires specific GitHub API scopes to function correctly. This document details the required permissions.

## Required Scopes

### Personal Access Token (Classic)

If using a classic personal access token, grant these scopes:

| Scope | Description | Used For |
|-------|-------------|----------|
| `repo` | Full control of private repositories | Access to PRs, comments, threads |
| `read:user` | Read user profile data | Author information |
| `user:email` | Read user email addresses | (Optional) User identification |

**Minimum scope:** `public_repo` for public repositories only.

### Fine-Grained Personal Access Token

For fine-grained tokens, configure these permissions:

#### Repository Permissions

| Permission | Access | Description |
|------------|--------|-------------|
| Pull requests | Read & Write | View PRs, post comments, resolve threads |
| Contents | Read | Access repository files |
| Metadata | Read | Basic repository information |

#### Account Permissions

| Permission | Access | Description |
|------------|--------|-------------|
| Contents | Read | Repository metadata |

## OAuth App (gh CLI)

When using `gh auth login`, the GitHub CLI requests these scopes automatically:

- `repo` (for private repos)
- `read:user`
- `user:email`

**No additional configuration needed** if using `gh` for authentication.

## Verification

### Check Token Scopes

```sh
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
     -H "Accept: application/vnd.github.v3+json" \
     https://api.github.com/user
```

Check the `X-OAuth-Scopes` header in the response.

### Test Permissions

Try listing PRs:

```sh
bv prs
```

If you see "403 Forbidden" or "bad credentials", your token may lack required scopes.

## Enterprise GitHub

For GitHub Enterprise Server:

1. Same scopes apply
2. Ensure your instance supports GraphQL API
3. Rate limits may differ

## Organization Policies

If your organization has restrictions:

1. **SSO Authorization:** Ensure token is SSO-enabled for the organization
   ```sh
   gh auth refresh -h github.com
   ```

2. **IP Allow Lists:** Ensure your IP is allowed

3. **Required Status Checks:** May affect PR visibility

## Troubleshooting

### "403 Forbidden" on PR operations

**Cause:** Missing `repo` scope

**Solution:** Regenerate token with `repo` scope

### "403 Forbidden" on private repos

**Cause:** Token lacks private repo access

**Solution:** Enable `repo` scope (not just `public_repo`)

### "Resource not accessible by integration"

**Cause:** Token not authorized for organization

**Solution:** Authorize token for organization SSO

## Security Best Practices

1. **Use fine-grained tokens** when possible (more restrictive)
2. **Rotate tokens regularly** (every 90 days recommended)
3. **Never commit tokens** to version control
4. **Use environment variables** or secret managers
5. **Revoke unused tokens** in GitHub settings

## Token Storage

bad-vibes stores tokens in:
- Location: `~/.cache/bad-vibes/token`
- Permissions: `0600` (owner read/write only)
- TTL: 1 hour (re-cached from `gh` or env)

## Rate Limits

| Type | Limit | Reset |
|------|-------|-------|
| REST (authenticated) | 5,000/hour | Rolling window |
| GraphQL (authenticated) | 5,000 points/hour | Fixed hour |

**Note:** GraphQL mutations (resolve thread, post comment) cost more points than queries.

### Check Rate Limit

```sh
bv --help  # If rate limit command is implemented
# Or
curl -H "Authorization: Bearer $GITHUB_TOKEN" \
     https://api.github.com/rate_limit
```

## Additional Resources

- [GitHub Token Documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens)
- [GitHub GraphQL API Docs](https://docs.github.com/en/graphql)
- [GitHub REST API Docs](https://docs.github.com/en/rest)
