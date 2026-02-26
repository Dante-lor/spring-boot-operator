#!/bin/sh
set -e

# Install Operator SDK
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
export OS=$(uname | awk '{print tolower($0)}')

export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.42.0
curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}

# Verify the install
gpg --keyserver keyserver.ubuntu.com --recv-keys 052996E2A20B5C7E

curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt
curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt.asc
gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify checksums.txt.asc

grep operator-sdk_${OS}_${ARCH} checksums.txt | sha256sum -c -
chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk

# KubeBuilder
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/linux/amd64
chmod +x kubebuilder
sudo mv kubebuilder /usr/local/bin/

# zensical (Docs)
echo 'alias zensical="docker run --rm -it -p 8000:8000  \
    -v ${LOCAL_WORKSPACE_FOLDER}/mkdocs.yml:/docs/mkdocs.yml \
    -v ${LOCAL_WORKSPACE_FOLDER}/docs:/docs/docs \
    zensical/zensical"' >> ~/.bashrc


