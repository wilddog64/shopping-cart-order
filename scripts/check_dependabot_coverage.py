#!/usr/bin/env python3
"""Fail when a tracked dependency manifest lacks Dependabot coverage."""

import fnmatch
from pathlib import PurePosixPath
import subprocess
import sys

import yaml


def required_updates(paths):
    required = {("github-actions", "/")}
    manifests = {"go.mod": "gomod", "pom.xml": "maven", "package.json": "npm"}
    for path in paths:
        path = PurePosixPath(path)
        ecosystem = manifests.get(path.name)
        if path.name == "Dockerfile" or path.name.startswith("Dockerfile."):
            ecosystem = "docker"
        if ecosystem:
            directory = "/" if str(path.parent) == "." else "/" + str(path.parent)
            required.add((ecosystem, directory))
    return required


def missing_updates(paths, config):
    updates = config.get("updates", [])
    missing = []
    for ecosystem, directory in sorted(required_updates(paths)):
        # Updates targeting another branch do not cover the default branch.
        covered = any(
            update.get("package-ecosystem") == ecosystem
            and not update.get("target-branch")
            and any(
                fnmatch.fnmatchcase(directory, pattern.rstrip("/") or "/")
                for pattern in (
                    update.get("directories", [])
                    + ([update["directory"]] if "directory" in update else [])
                )
            )
            for update in updates
        )
        if not covered:
            missing.append((ecosystem, directory))
    return missing


def main():
    paths = subprocess.check_output(["git", "ls-files", "-z"], text=True).split("\0")
    with open(".github/dependabot.yml", encoding="utf-8") as source:
        config = yaml.safe_load(source)
    missing = missing_updates(filter(None, paths), config)
    for ecosystem, directory in missing:
        print(f"Missing Dependabot coverage: {ecosystem} {directory}", file=sys.stderr)
    if missing:
        return 1
    print("All tracked dependency manifests have Dependabot coverage.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
