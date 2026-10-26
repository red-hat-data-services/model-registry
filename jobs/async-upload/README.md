# Kubeflow Model Registry – Async Upload Job

> Lightweight, non‑root Python Job containerised for **transferring model artefacts** between storage back‑ends (S3, OCI, PVC, …) and registering them in **Kubeflow Model Registry**. Born out of the discussion in [kubeflow/model-registry #1108](https://github.com/kubeflow/model-registry/issues/1108).

---

## Quick start

```bash
# 1 – Build & tag ( reproducible build args are optional )
docker build \
  --build-arg VCS_REF=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t quay.io/<org>/async-upload-job:$(git rev-parse --short HEAD) .

# 2 – Push to your registry
docker push quay.io/<org>/model-registry-job-async-upload:<tag>

# TODO: Run locally...
```

## Updating Python dependencies

Dependencies are declared in `pyproject.toml` and resolved in `poetry.lock`.
The standard container installs from the generated `requirements.txt`; downstream
Konflux builds also use `requirements-downstream.txt` and `requirements-build.txt`.

Run the following from this directory (`cd jobs/async-upload` from the repository
root). Use Poetry 2.x and Python 3.12 to match the container's Python version:

```bash
poetry env use python3.12

# Update a package within its existing version constraints (requests is an example).
poetry update requests --lock

# Install the locked dependencies and regenerate requirements.txt.
make install
```

To select a version outside the current constraint, first edit the package's
constraint in `pyproject.toml`, then run `poetry update <package> --lock` and
`make install`. Keep development dependencies in the `dev` group and integration
test dependencies in the `integration` group. Preserve any required extras, such
as `boto3` and `olot` on `model-registry`.

The same update command works for transitive dependencies, for example
`poetry update urllib3 --lock`. If another package's constraints prevent the
desired version, update that package too. Omit the package name only when you
intend to update all dependencies. See the [Poetry update documentation](https://python-poetry.org/docs/cli/#update)
for details.

`make install` installs the Poetry export plugin, installs dependencies, and
exports the runtime requirements with the generated-file header. Do not edit
`poetry.lock` or `requirements.txt` manually. CI reruns `make install` and checks
that generated files remain unchanged.

### Downstream / Konflux requirements

After exporting runtime dependency changes:

1. Update duplicate pins in `requirements-downstream.txt` to match
   `requirements.txt`, including environment markers. Keep the
   `-r requirements.txt` include and downstream-only dependencies such as
   `puccinialin==0.1.5`. Add or remove transitive dependencies as needed; all
   downstream runtime dependencies must be explicitly pinned for disconnected
   builds.
2. Regenerate the source-build requirements using
   [pybuild-deps](https://pybuild-deps.readthedocs.io/en/latest/usage.html#compile)
   under Python 3.12. With `pipx` installed:

   ```bash
   pipx install --python python3.12 pybuild-deps
   pybuild-deps compile --output-file=requirements-build.txt requirements-downstream.txt
   ```

   Review the generated pins, including any conflicting versions. Konflux
   prefetches all three requirements files with binary distributions disabled,
   so newly introduced source-build dependencies must be included.

### Validate and submit

```bash
poetry check --lock
make test
git diff --check
git diff -- pyproject.toml poetry.lock requirements.txt requirements-downstream.txt requirements-build.txt
```

Review and commit the changed dependency files together. For container or storage
integration changes, also run `make test-e2e` or `make test-integration`; these
targets require Docker, KinD, and the associated cluster setup described in the
[integration test README](tests/integration/README.md).

## Job Configuration

The async job is able to take a number of configuration parameters and environment variables which can be consumed to perform the job of synchronizing a model from a source to a destination.

When using environment variables to configure the job, you will need to provide them in the Kubernetes Job manifest in the `spec.template.spec.containers[*].env` section. From the job's perspective, these will become standard env vars that it can read. See the [samples directory](./samples/) for typical usage.

When using a parameter-based approach, the configuration variables will need to be passed in as `args` to the `command`.

When providing parameters in a mixed-fashion to the job, the job will prioritize certain sources of those parameters over others. The order of priority is below:

1. Command-line arguments (`args: []` in the manifest)
2. Environment variables (`env: { ... }` in the manifest)
3. Credentials files (read from the parameter \*\_CREDENTIALS_PATH env/arg)
4. Default values

Below is a table of the configuration that can be passed into the job

See asterisks below table for details

| Environment Variable                         | Arg                                 | Default Value     | Required | Description                                                                                            |
| -------------------------------------------- | ----------------------------------- | ----------------- | -------- | ------------------------------------------------------------------------------------------------------ |
| MODEL_SYNC_SOURCE_TYPE                       | --source-type                       | s3                | ✅       |                                                                                                        |
| MODEL_SYNC_SOURCE_S3_CREDENTIALS_PATH        | --source-s3-credentials-path        |                   |          |                                                                                                        |
| MODEL_SYNC_SOURCE_OCI_CREDENTIALS_PATH       | --source-oci-credentials-path       |                   |          |                                                                                                        |
| MODEL_SYNC_SOURCE_URI_CREDENTIALS_PATH       | --source-uri-credentials-path       |                   |          |                                                                                                        |
| MODEL_SYNC_SOURCE_URI                        | --source-uri                        |                   | ✅\#     | When --source-type is "uri". The URI to download from                                                  |
| MODEL_SYNC_SOURCE_AWS_BUCKET                 | --source-aws-bucket                 |                   | ✅\*     | When --source-type is "s3"                                                                             |
| MODEL_SYNC_SOURCE_AWS_KEY                    | --source-aws-key                    |                   | ✅\*     | "                                                                                                      |
| MODEL_SYNC_SOURCE_AWS_REGION                 | --source-aws-region                 |                   |          | "                                                                                                      |
| MODEL_SYNC_SOURCE_AWS_ACCESS_KEY_ID          | --source-aws-access-key-id          |                   | ✅\*     | "                                                                                                      |
| MODEL_SYNC_SOURCE_AWS_SECRET_ACCESS_KEY      | --source-aws-secret-access-key      |                   | ✅\*     | "                                                                                                      |
| MODEL_SYNC_SOURCE_AWS_ENDPOINT               | --source-aws-endpoint               |                   |          | "                                                                                                      |
| MODEL_SYNC_SOURCE_OCI_URI                    | --source-oci-uri                    |                   | ✅\+     | When --source-type is "oci". The tag to use when pulling the image                                     |
| MODEL_SYNC_SOURCE_OCI_REGISTRY               | --source-oci-registry               |                   | ✅\+     | When --source-type is "oci". Indicates which registry the creds belong to                              |
| MODEL_SYNC_SOURCE_OCI_USERNAME               | --source-oci-username               |                   | ✅\+     | "                                                                                                      |
| MODEL_SYNC_SOURCE_OCI_PASSWORD               | --source-oci-password               |                   | ✅\+     | "                                                                                                      |
| MODEL_SYNC_DESTINATION_TYPE                  | --destination-type                  | oci               | ✅       |                                                                                                        |
| MODEL_SYNC_DESTINATION_S3_CREDENTIALS_PATH   | --destination-s3-credentials-path   |                   |          |                                                                                                        |
| MODEL_SYNC_DESTINATION_OCI_CREDENTIALS_PATH  | --destination-oci-credentials-path  |                   |          |                                                                                                        |
| MODEL_SYNC_DESTINATION_AWS_BUCKET            | --destination-aws-bucket            |                   | ✅\*     | When --destination-type is "s3"                                                                        |
| MODEL_SYNC_DESTINATION_AWS_KEY               | --destination-aws-key               |                   | ✅\*     | "                                                                                                      |
| MODEL_SYNC_DESTINATION_AWS_REGION            | --destination-aws-region            |                   |          | "                                                                                                      |
| MODEL_SYNC_DESTINATION_AWS_ACCESS_KEY_ID     | --destination-aws-access-key-id     |                   | ✅\*     | "                                                                                                      |
| MODEL_SYNC_DESTINATION_AWS_SECRET_ACCESS_KEY | --destination-aws-secret-access-key |                   | ✅\*     | "                                                                                                      |
| MODEL_SYNC_DESTINATION_AWS_ENDPOINT          | --destination-aws-endpoint          |                   |          | "                                                                                                      |
| MODEL_SYNC_DESTINATION_OCI_URI               | --destination-oci-uri               |                   | ✅\+     | When --destination-type is "oci". The tag to use when pushing the image                                |
| MODEL_SYNC_DESTINATION_OCI_REGISTRY          | --destination-oci-registry          |                   | ✅\+     | When --destination-type is "oci". Indicates which registry the creds belong to                         |
| MODEL_SYNC_DESTINATION_OCI_USERNAME          | --destination-oci-username          |                   | ✅\+     | "                                                                                                      |
| MODEL_SYNC_DESTINATION_OCI_PASSWORD          | --destination-oci-password          |                   | ✅\+     | "                                                                                                      |
| MODEL_SYNC_DESTINATION_OCI_BASE_IMAGE        | --destination-oci-base-image        | busybox:latest    |          | When --destination-type is "oci". The image to use when pushing to an OCI registry                     |
| MODEL_SYNC_DESTINATION_OCI_ENABLE_TLS_VERIFY | --destination-oci-enable-tls-verify | true              |          | When --destination-type is "oci". Specifies whether to use TLS when pushing to registry                |
| MODEL_SYNC_MODEL_UPLOAD_INTENT               | --model-upload-intent               | update_artifact   |          | The intent of the upload job. Options include `["create_model", "create_version", "update_artifact"]`. |
| MODEL_SYNC_MODEL_ID                          | --model-id                          |                   | ✅\%     | The `RegisteredModel.id`                                                                               |
| MODEL_SYNC_MODEL_VERSION_ID                  | --model-version-id                  |                   | ✅\%     | The `ModelVersion.id`                                                                                  |
| MODEL_SYNC_MODEL_ARTIFACT_ID                 | --model-artifact-id                 |                   | ✅\%     | The `ModelArtifact.id`                                                                                 |
| MODEL_SYNC_STORAGE_PATH                      | --storage-path                      | `/tmp/model-sync` | ✅       | Temporary storage, must be large enough to hold the entire model                                       |
| MODEL_SYNC_REGISTRY_SERVER_ADDRESS           | --registry-server-address           |                   | ✅       | Server address for the model-registry client to connect to                                             |
| MODEL_SYNC_REGISTRY_PORT                     | --registry-port                     | 443               |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_IS_SECURE                | --registry-is-secure                | True              |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_AUTHOR                   | --registry-author                   |                   |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_USER_TOKEN               | --registry-user-token               |                   |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_USER_TOKEN_ENVVAR        | --registry-user-token-envvar        |                   |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_CUSTOM_CA                | --registry-custom-ca                |                   |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_CUSTOM_CA_ENVVAR         | --registry-custom-ca-envvar         |                   |          |                                                                                                        |
| MODEL_SYNC_REGISTRY_LOG_LEVEL                | --registry-log-level                |                   |          |                                                                                                        |

✅\*: Must be present in some form when the source/destination is `s3`. This might be from the parameter in the table, or from the credentials file(s) that was specified/provided.

✅\+: Must be present in some from when the source/destination is `oci`. This might be from the parameter in the table, or from the credentials file(s) that was specified/provided.

✅\#: Must be present in some form when the source is `uri`. This might be from the parameter in the table, or from the credentials file(s) that was specified/provided.

✅\%: Must be present in some form depending on what the `model-upload-intent` is set to.

## Upload Intent Types

The job supports different intent types that determine how it interacts with the Model Registry:

### `update_artifact`

(Default)

Updates an existing ModelArtifact's URI and sets it to LIVE state.
- **Required**: `MODEL_SYNC_MODEL_ARTIFACT_ID`
- **ConfigMap**: Not required

### `create_model`

Creates a new RegisteredModel, ModelVersion, and ModelArtifact.
- **Required**: `MODEL_SYNC_METADATA_CONFIGMAP_PATH` with complete model metadata
- **ConfigMap**: Must contain RegisteredModel, ModelVersion, and ModelArtifact metadata

### `create_version`

Creates a new ModelVersion and ModelArtifact under an existing RegisteredModel.
- **Required**: `MODEL_SYNC_MODEL_ID` and `MODEL_SYNC_METADATA_CONFIGMAP_PATH`
- **ConfigMap**: Must contain ModelVersion and ModelArtifact metadata

## ConfigMap Usage

For `create_model` and `create_version` intents, provide metadata via a mounted ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: model-metadata
data:
  # RegisteredModel fields (create_model only)
  RegisteredModel.name: "my-model"
  RegisteredModel.description: "Model description"
  RegisteredModel.owner: "data-science-team"
  RegisteredModel.custom_properties: |
    {"project": "sentiment-analysis", "team": "nlp"}
  # ModelVersion fields
  ModelVersion.name: "1.0.0"
  ModelVersion.description: "Initial release"
  ModelVersion.author: "Jane Doe"
  ModelVersion.custom_properties: |
    {"accuracy": 0.95, "f1_score": 0.93}
  # ModelArtifact fields
  ModelArtifact.name: "sentiment-analyzer"
  ModelArtifact.model_format_name: "tensorflow"
  ModelArtifact.model_format_version: "2.8"
  ModelArtifact.storage_key: "s3-connection"
  ModelArtifact.custom_properties: |
    {"model_size_mb": 438, "inference_time_ms": 120}
```

Mount the ConfigMap in your Job:

```yaml
spec:
  template:
    spec:
      containers:
      - name: async-upload
        env:
        - name: MODEL_SYNC_MODEL_UPLOAD_INTENT
          value: "create_model"
        - name: MODEL_SYNC_METADATA_CONFIGMAP_PATH
          value: "/etc/model-metadata"
        volumeMounts:
        - name: metadata
          mountPath: /etc/model-metadata
          readOnly: true
      volumes:
      - name: metadata
        configMap:
          name: model-metadata
```

## References

- Issue thread : [https://github.com/kubeflow/model-registry/issues/1108](https://github.com/kubeflow/model-registry/issues/1108)
- OCI Image Spec : [https://github.com/opencontainers/image-spec](https://github.com/opencontainers/image-spec)
- Kubernetes Pod Security : [https://kubernetes.io/docs/concepts/security/pod-security-standards/](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
