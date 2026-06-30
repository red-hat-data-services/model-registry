#!/bin/bash
# Builds the async upload job hermetically using podman. The hermeto CLI expects to be ran from the git root
set -euxo pipefail

if [ $# -ne 2 ]; then
  echo "Usage: $0 <activation-key> <org-id>" >&2
  echo "Create an activation key here https://console.redhat.com/insights/connector/activation-keys"
  exit 1
fi

ACTIVATION_KEY="$1"
ORG_ID="$2"

shopt -s expand_aliases
mkdir -p /tmp/cachi2
alias hermeto='podman run --rm -ti -v shared-certs:/certs -v "$PWD:$PWD:z" -w "$PWD" -v /tmp/cachi2:/tmp/cachi2:z ghcr.io/hermetoproject/hermeto:latest'


cleanup() { podman exec register-container subscription-manager unregister 2>/dev/null; podman stop register-container 2>/dev/null; podman volume rm shared-certs 2>/dev/null; }
trap cleanup EXIT

podman run -d --rm --init --name register-container -v shared-certs:/certs registry.access.redhat.com/ubi9/ubi:9.6 sleep infinity
podman exec -i register-container /bin/bash <<EOF
    set -e
    subscription-manager config --rhsm.manage_repos=0
    subscription-manager register --activationkey="${ACTIVATION_KEY}" --org="${ORG_ID}"
    cp \$(ls /etc/pki/entitlement/*.pem | grep -v '\-key\.pem$') /certs/cert.pem
    cp \$(ls /etc/pki/entitlement/*-key.pem) /certs/key.pem
    cp /etc/rhsm/ca/redhat-uep.pem /certs/redhat-uep.pem
EOF

hermeto fetch-deps --output /tmp/cachi2/output \
  '[{
    "type": "rpm",
    "path": "jobs/async-upload",
    "options": {
      "ssl": {
        "client_cert": "/certs/cert.pem",
        "client_key": "/certs/key.pem",
        "ca_bundle": "/certs/redhat-uep.pem"
      }
    }
  },
  {
    "type": "pip",
    "path": "jobs/async-upload",
    "requirements_files": ["requirements-aipcc.txt"],
    "binary": { "arch": "x86_64", "os": "linux" }
  }]'

hermeto inject-files /tmp/cachi2/output --for-output-dir /cachi2/output
hermeto generate-env -f env -o /tmp/cachi2/cachi2.env --for-output-dir /tmp/cachi2 /tmp/cachi2/output

podman build --no-cache -f jobs/async-upload/Dockerfile.konflux jobs/async-upload/ \
  --volume /tmp/cachi2:/cachi2:z \
  --volume /tmp/cachi2/output/deps/rpm/x86_64/repos.d:/etc/yum.repos.d:z \
  --network none \
  --env CARGO_HOME=/cachi2/output/.cargo \
  --env PIP_FIND_LINKS=/cachi2/output/deps/pip \
  --env PIP_NO_INDEX=true \
  --build-arg TARGETARCH=x86_64 \
  --arch x86_64 \
  --tag async-job
