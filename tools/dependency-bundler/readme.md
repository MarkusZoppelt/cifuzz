# Dependency Bundler

This is a small Linux/macOS script to download and package all CI Fuzz Java dependencies.

Downloaded artifacts are cached in `/tmp/ci-coursier`, make sure to clean them up if you run
the script with different versions.

## Usage

```bash
./bundle-dependencies.sh <repository_user> <repository_password> [--clean]
```
