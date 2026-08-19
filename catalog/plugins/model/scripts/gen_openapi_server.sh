#!/usr/bin/env bash

set -e

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}

PROJECT_ROOT=$(realpath "$(dirname "$0")/../../..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")

VERSIONS=("v1alpha1" "v1")
if [[ -n "$1" ]]; then
    VERSIONS=("$1")
fi

py-re-replace() {
  python3 -c "
import fileinput, re, sys
count, pattern, replacement, filepaths = int(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4:]
for filepath in filepaths:
    for line in fileinput.FileInput(filepath, inplace=True, backup=''):
        sys.stdout.write(re.sub(pattern, replacement, line, count=count))
" "$@"
}

apply-model-delegation-patch() {
  python3 -c "
import sys
filepath = sys.argv[1]
with open(filepath, 'r') as f:
    content = f.read()

target = 'modelNameParam := chi.URLParam(r, \"*\")\n\tif modelNameParam == \"\" {\n\t\tc.errorHandler(w, r, &RequiredError{\"*\"}, nil)\n\t\treturn\n\t}'
patch = '''
	// Special handling for getModel to delegate /artifacts requests to getAllModelArtifacts
	// The wildcard /* pattern catches /artifacts requests, but we want those to go to GetAllModelArtifacts
	if strings.HasSuffix(r.URL.Path, \"/artifacts\") {
		// Extract the model name by removing the /artifacts suffix
		modelName := strings.TrimSuffix(modelNameParam, \"/artifacts\")

		// Add the model_name parameter to the route context so GetAllModelArtifacts can access it
		chi.RouteContext(r.Context()).URLParams.Add(\"model_name\", modelName)

		// Call the GetAllModelArtifacts handler directly
		c.GetAllModelArtifacts(w, r)
		return
	}

	// Same for /artifacts/performance requests to getAllModelPerformanceArtifacts
	if strings.HasSuffix(r.URL.Path, \"/artifacts/performance\") {
		modelName := strings.TrimSuffix(modelNameParam, \"/artifacts/performance\")
		chi.RouteContext(r.Context()).URLParams.Add(\"model_name\", modelName)
		c.GetAllModelPerformanceArtifacts(w, r)
		return
	}'''

if target in content and patch not in content:
    content = content.replace(target, target + '\n' + patch, 1)
    with open(filepath, 'w') as f:
        f.write(content)
" "$1"
}

TMPFILES=()
trap 'rm -rf "${TMPFILES[@]}"' EXIT

for VER in "${VERSIONS[@]}"; do
    echo "Generating model plugin server stubs ($VER)"
    DST="$PROJECT_ROOT/internal/server/openapi/$VER"
    mkdir -p "$DST"

    SPEC=$(mktemp -t model_plugin_spec_XXXXXX.yaml)
    GENDIR=$(mktemp -d -t model_openapi_gen_XXXXXX)
    TMPFILES+=("$SPEC" "$GENDIR")

    "$REPO_ROOT/scripts/assemble_plugin_spec.sh" model "$SPEC" "$VER"

    "$OPENAPI_GENERATOR" generate \
        -i "$SPEC" -g go-server -o "$GENDIR" --package-name "$VER" \
        --additional-properties=outputAsLibrary=true,enumClassPrefix=true,router=chi,sourceFolder=,onlyInterfaces=true,isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true \
        --template-dir "$REPO_ROOT/templates/go-server"

    # Fix array type references in temp files
    py-re-replace 0 'model\.\[\]ArtifactTypeQueryParam' '[]model.ArtifactTypeQueryParam' "$GENDIR/api.go"
    py-re-replace 0 'model\.\[\]ArtifactType2QueryParam' '[]model.ArtifactTypeQueryParam' "$GENDIR/api.go"

    # Fix package imports in temp files
    py-re-replace 1 'github\.com/kubeflow/hub/pkg/openapi' 'github.com/kubeflow/hub/catalog/pkg/openapi' \
        "$GENDIR/api_model_catalog_service.go" \
        "$GENDIR/api.go"

    # Fix wildcard path placeholder
    py-re-replace 1 '\{model_name\+\}|model_name\+' '*' "$GENDIR/api_model_catalog_service.go"

    # Copy this plugin's files to the shared output directory
    cp "$GENDIR/api_model_catalog_service.go" "$DST/"
    cp "$GENDIR/api.go" "$DST/api_model.go"

    # Copy shared infrastructure
    cp "$GENDIR"/impl.go "$GENDIR"/error.go "$GENDIR"/helpers.go "$GENDIR"/routers.go "$GENDIR"/logger.go "$DST/" 2>/dev/null || true

    # Copy model type files — needed by gen_type_asserts.sh
    cp "$GENDIR"/model_*.go "$DST/" 2>/dev/null || true

    apply-model-delegation-patch "$DST/api_model_catalog_service.go"

    "$REPO_ROOT/bin/goimports" -w "$DST/api_model_catalog_service.go" "$DST/api_model.go"

    echo "Model plugin server stubs generated ($VER)"
done
