"""Regression checks for language migrations and directory coverage."""

import unittest

from check_dependabot_coverage import missing_updates


class CoverageTests(unittest.TestCase):
    def config(self, *entries):
        return {"updates": [{"package-ecosystem": "github-actions", "directory": "/"}, *entries]}

    def test_go_rewrite_requires_go_module_and_docker_coverage(self):
        config = self.config(
            {"package-ecosystem": "maven", "directory": "/"},
            {"package-ecosystem": "docker", "directory": "/"},
        )
        self.assertEqual(
            missing_updates(["pom.xml", "Dockerfile", "go/go.mod", "go/Dockerfile"], config),
            [("docker", "/go"), ("gomod", "/go")],
        )

    def test_matching_ecosystem_in_wrong_directory_is_not_coverage(self):
        self.assertEqual(
            missing_updates(["go/go.mod"], self.config({"package-ecosystem": "gomod", "directory": "/"})),
            [("gomod", "/go")],
        )

    def test_future_npm_manifest_requires_coverage(self):
        self.assertEqual(missing_updates(["web/package.json"], self.config()), [("npm", "/web")])

    def test_plural_directories_and_globs_cover_new_modules(self):
        config = self.config({"package-ecosystem": "gomod", "directories": ["/go", "/services/*"]})
        self.assertEqual(missing_updates(["go/go.mod", "services/payments/go.mod"], config), [])

    def test_non_default_branch_does_not_cover_default_branch(self):
        config = self.config({"package-ecosystem": "gomod", "directory": "/go", "target-branch": "release"})
        self.assertEqual(missing_updates(["go/go.mod"], config), [("gomod", "/go")])

    def test_complete_coverage_passes(self):
        config = self.config(
            {"package-ecosystem": "gomod", "directory": "/go"},
            {"package-ecosystem": "docker", "directories": ["/", "/go"]},
            {"package-ecosystem": "maven", "directory": "/"},
        )
        self.assertEqual(missing_updates(["pom.xml", "Dockerfile", "go/go.mod", "go/Dockerfile.local"], config), [])


if __name__ == "__main__":
    unittest.main()
