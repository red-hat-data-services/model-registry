#!/bin/bash
# Builds the async upload job hermetically using podman. The hermeto CLI expects to be ran from the git root
set -euxo pipefail
shopt -s expand_aliases
mkdir -p /tmp/cachi2
alias hermeto='podman run --rm -ti -v "$PWD:$PWD:z" -w "$PWD" -v /tmp/cachi2:/tmp/cachi2:z ghcr.io/hermetoproject/hermeto:latest'

hermeto fetch-deps --output /tmp/cachi2/output '[{"type": "rpm", "path": "jobs/async-upload"},{
  "type": "pip", "path": "jobs/async-upload",
  "requirements_files": ["requirements.txt"],
  "requirements_build_files": ["requirements-build.txt","requirements-extra-build-deps.txt"],
  "binary": { "arch": "x86_64", "os": "linux" }
}]'

hermeto inject-files /tmp/cachi2/output --for-output-dir /cachi2/output
hermeto generate-env -f env -o /tmp/cachi2/cachi2.env --for-output-dir /tmp/cachi2 /tmp/cachi2/output

podman build -f jobs/async-upload/Dockerfile.konflux jobs/async-upload/ \
  --volume /tmp/cachi2:/cachi2:z \
  --volume /tmp/cachi2/output/deps/rpm/x86_64/repos.d:/etc/yum.repos.d:z \
  --network none \
  --env PIP_NO_BINARY=:all: \
  --env CARGO_HOME=/cachi2/output/.cargo \
  --env PIP_FIND_LINKS=/cachi2/output/deps/pip \
  --env PIP_NO_INDEX=true \
  --build-arg TARGETARCH=x86_64 \
  --arch x86_64 \
  --tag async-job
