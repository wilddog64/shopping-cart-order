# OWASP Dependency Check fails without NVD API key

## Summary

After adding the OWASP Dependency Check plugin (per p4-linter spec), the `mvn verify`
step now fails in CI because the plugin cannot download NVD data without an API key.
The GitHub Actions logs show repeated `UpdateException: Error updating the NVD Data;
the NVD returned a 403 or 404 error` followed by `NoDataException: No documents exist`.
Attempts to provide the public `DEMO_KEY` and to disable `autoUpdate` both failed —
NVD now requires an authenticated API key even for first-time database creation.

## Impact

- CI run `23095750831` on branch `feature/p4-linter` fails in the `Build & Test` job,
  blocking the Checkstyle/OWASP PR from passing.
- Checkstyle succeeds; only dependency-check is failing before any CVE scan happens,
  so no vulnerability signal is produced.

## Next Steps

- Provision an `NVD_API_KEY` secret in the `wilddog64/shopping-cart-order` repository
  (or appropriate GitHub environment) with a valid key from https://nvd.nist.gov/developers.
- Once the secret exists, the workflow already passes it to Maven via the
  `NVD_API_KEY` env var and `pom.xml` configuration (`<nvdApiKey>${env.NVD_API_KEY}</nvdApiKey>`).
- Re-run the workflow (push or `gh run rerun`) to populate the dependency-check
  data store and unblock the linter PR.

Until the API key is available, OWASP dependency-check will continue to fail before
any dependency analysis runs.
