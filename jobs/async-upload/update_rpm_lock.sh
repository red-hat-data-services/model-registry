#!/bin/bash
set -e


if [ $# -ne 2 ]; then
  echo "Usage: $0 <activation-key> <org-id>" >&2
  echo "Create an activation key here https://console.redhat.com/insights/connector/activation-keys"
  exit 1
fi

ACTIVATION_KEY="$1"
ORG_ID="$2"

podman run --rm -v ${PWD}:/work -w /work -i \
-v ${XDG_RUNTIME_DIR}/containers/auth.json:/run/containers/0/auth.json:ro \
registry.access.redhat.com/ubi9/ubi:9.6 /bin/bash <<EOF
    set -e
    subscription-manager config --rhsm.manage_repos=0
    subscription-manager register --activationkey="${ACTIVATION_KEY}" --org="${ORG_ID}"
    dnf install skopeo -y
    python3 -m ensurepip --default-pip
    python3 -m pip install https://github.com/konflux-ci/rpm-lockfile-prototype/archive/refs/tags/v0.23.0.tar.gz
    DNF_VAR_SSL_CLIENT_CERT=\$(ls /etc/pki/entitlement/*.pem | grep -v '\-key\.pem$') \
    DNF_VAR_SSL_CLIENT_KEY=\$(ls /etc/pki/entitlement/*-key.pem) \
    rpm-lockfile-prototype rpms.in.yaml
    subscription-manager  unregister
EOF
