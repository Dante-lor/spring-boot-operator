#!/bin/bash

VERSION=$1

./bin/opm render ghcr.io/dante-lor/spring-boot-operator-bundle:v$VERSION --output yaml > catalog/spring-boot-operator/v$VERSION.yaml

echo " - name: spring-boot-operator.v${VERSION}" >> catalog/spring-boot-operator/channel.yaml