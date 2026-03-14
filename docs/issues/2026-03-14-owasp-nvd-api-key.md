# OWASP Dependency Check fails without NVD API key

## Summary

After enabling Checkstyle + OWASP in CI, the `mvn verify` step now fails because
OWASP Dependency Check cannot download NVD data without an API key. The latest run
(`23096292362`) throws `UpdateException: Error updating the NVD Data; the NVD
returned a 403 or 404 error` and `NoDataException: No documents exist`.

## Impact

- CI for branch `feature/p4-linter` (PR #2) fails in the build job, blocking the long-term linter gate.
- No vulnerability scan results are produced until the NVD feed can be downloaded.

## Workaround / Next Steps

1. Add a valid `NVD_API_KEY` secret to the repository settings (per OWASP Dependency Check
   recommendation). Instructions: https://github.com/jeremylong/DependencyCheck?tab=readme-ov-file#nvd-api-key-highly-recommended.
2. (Optional) Prime the GH runner cache by running dependency-check once with the key so the
   local data directory is populated.
3. Re-run the workflow (`gh run rerun 23096292362`) after the secret is available; the existing
   workflow already exports `NVD_API_KEY` to Maven so no code changes are needed.
4. Until the API key exists, OWASP will fail before scanning dependencies, so the Checkstyle/OWASP
   PR cannot merge.

Last failure: run `23096292362`, commit `bea10f4`, message "Re-enable OWASP failure mode".
