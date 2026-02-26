#!/bin/bash

VERSION=$1

CHANNEL_FILE="catalog/spring-boot-operator/channel.yaml"
PACKAGE="spring-boot-operator"
IMAGE="ghcr.io/dante-lor/${PACKAGE}-bundle:v${VERSION}"

if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "Rendering bundle ${IMAGE}..."
./bin/opm render ${IMAGE} --output yaml > catalog/spring-boot-operator/v${VERSION}.yaml

# Extract current head (last entry in channel)
PREVIOUS=$(grep "^  - name:" ${CHANNEL_FILE} | tail -n 1 | awk '{print $3}')

if [ -z "$PREVIOUS" ]; then
  echo "No previous version found, creating first channel entry..."
  echo "  - name: ${PACKAGE}.v${VERSION}" >> ${CHANNEL_FILE}
else
  echo "Previous head detected: ${PREVIOUS}"
  echo "Adding ${VERSION} replacing ${PREVIOUS}..."
  cat <<EOF >> ${CHANNEL_FILE}
  - name: ${PACKAGE}.v${VERSION}
    replaces: ${PREVIOUS}
EOF
fi

echo "Alpha channel updated successfully."