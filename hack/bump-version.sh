#!/bin/bash

# Old version is first argument
# It is the version of the software that was just released
RELEASED_VERSION=$1

# Update the docs so that our versions are now correct
find docs -name "*.md" | xargs sed -i -E "s/[0-9]+\.[0-9]+\.[0-9]+/$RELEASED_VERSION/g"

MAJOR=$(echo "$RELEASED_VERSION" | cut -d. -f1)
MINOR=$(echo "$RELEASED_VERSION" | cut -d. -f2)
PATCH=$(echo "$RELEASED_VERSION" | cut -d. -f3)

NEXT_PATCH=$((PATCH + 1))
NEXT_VERSION="${MAJOR}.${MINOR}.${NEXT_PATCH}"

# Update the version in the Makefile to the next version
sed -i "s/^VERSION ?= .*/VERSION ?= ${NEXT_VERSION}/" Makefile

# Update the version in the manager/kustomization file
sed -i -E "s/[0-9]+\.[0-9]+\.[0-9]+/$RELEASED_VERSION/g" config/manager/kustomization.yaml