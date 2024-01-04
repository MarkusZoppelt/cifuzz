#!/usr/bin/env bash

set -euf -o pipefail

if [[ -z $1 || -z $2 ]]; then
    echo "ERROR: Please provide repository username and password"
    exit 1
fi

if [ "${3-z}" == "--clean" ]; then
		echo "INFO: Cleaning up cached dependencies"
		rm -rf /tmp/ci-coursier
fi

COURSIER_JAR=/tmp/coursier.jar
COURSIER_CACHE=/tmp/ci-coursier
DEPENDENCIES_ARCHIVE=ci-dependencies.tar

REPOSITORY_USER=$1
REPOSITORY_PASSWORD=$2

if [[ ! -f "$COURSIER_JAR" ]]; then
    echo "INFO: Could not find \"$COURSIER_JAR\". Downloading..."
    curl -L 'https://github.com/coursier/coursier/releases/download/v2.1.8/coursier.jar' -o $COURSIER_JAR
    checksum=$(sha256sum $COURSIER_JAR | cut -d ' ' -f 1)
    if [ "$checksum" != "2b78bfdd3ef13fd1f42f158de0f029d7cbb1f4f652d51773445cf2b6f7918a87" ]; then
				echo "ERROR: Checksum of downloaded \"$COURSIER_JAR\" does not match"
				exit 1
		fi
fi

# Add private repository and credentials.
export COURSIER_REPOSITORIES="central|https://gitlab.code-intelligence.com/api/v4/projects/89/packages/maven"
export COURSIER_CREDENTIALS="gitlab.code-intelligence.com $REPOSITORY_USER:$REPOSITORY_PASSWORD"
export COURSIER_CACHE="$COURSIER_CACHE"

echo "INFO: Resolving dependencies"
java -jar $COURSIER_JAR fetch \
	"com.code-intelligence:jazzer-junit:0.23.0" \
	"com.code-intelligence:cifuzz-maven-extension:1.2.0" \
	"com.code-intelligence.cifuzz:com.code-intelligence.cifuzz.gradle.plugin:1.12.0"

echo "INFO: Packaging dependencies"
# Remove empty code-intelligence folders from repo1.maven.org cache,
# as CI artifacts are not published there anymore.
rm -rf $COURSIER_CACHE/https/repo1.maven.org/maven2/com/code-intelligence/

tar -cf $DEPENDENCIES_ARCHIVE -C $COURSIER_CACHE/https/repo1.maven.org/maven2/ --exclude="*__*" --exclude="*.checked" --exclude="*.sha1" .
tar -rf $DEPENDENCIES_ARCHIVE -C $COURSIER_CACHE/https/"${REPOSITORY_USER}"%40gitlab.code-intelligence.com/api/v4/projects/89/packages/maven/ --exclude="*__*" --exclude="*.checked" --exclude="*.sha1" .

echo "INFO: Saved dependencies at $DEPENDENCIES_ARCHIVE"
