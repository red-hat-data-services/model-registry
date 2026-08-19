PROJECT_BIN := $(CURDIR)/../bin
OPENAPI_GENERATOR := $(PROJECT_BIN)/openapi-generator-cli
GENQLIENT_BIN ?= $(PROJECT_BIN)/genqlient
GENQLIENT_CONFIG := internal/catalog/genqlient/genqlient.yaml
GENQLIENT_OUTPUT := internal/catalog/genqlient/generated.go
GENQLIENT_SOURCES := $(wildcard internal/catalog/genqlient/queries/*.graphql)
GRAPHQL_SCHEMA := internal/catalog/genqlient/queries/schema.graphql
GO ?= "$(shell which go)"

.PHONY: gen/openapi-server
gen/openapi-server:
	OPENAPI_GENERATOR="$(OPENAPI_GENERATOR)" ./scripts/gen_all_openapi_servers.sh

.PHONY: gen/openapi
gen/openapi: pkg/openapi/client.go

pkg/openapi/client.go: ../api/openapi/catalog-v1.yaml
	${OPENAPI_GENERATOR} generate \
		-i ../api/openapi/catalog-v1.yaml -g go -o pkg/openapi --package-name openapi \
		--ignore-file-override ./.openapi-generator-ignore --additional-properties=isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true,generateUnmarshalJSON=false
	$(PROJECT_BIN)/goimports -w pkg/openapi

.PHONY: gen/graphql
gen/graphql: $(GENQLIENT_OUTPUT)

$(GENQLIENT_OUTPUT): $(GENQLIENT_CONFIG) $(GENQLIENT_SOURCES) $(PROJECT_BIN)/genqlient
	$(GENQLIENT_BIN) --config $(GENQLIENT_CONFIG)

.PHONY: download/graphql-schema
download/graphql-schema:
	npx get-graphql-schema https://catalog.redhat.com/api/containers/graphql/ > $(GRAPHQL_SCHEMA)

.PHONY: clean-pkg-openapi
clean-pkg-openapi:
	while IFS= read -r file; do rm -f "pkg/openapi/$$file"; done < pkg/openapi/.openapi-generator/FILES

.PHONY: clean-graphql
clean-graphql:
	rm -f $(GENQLIENT_OUTPUT)

PLUGINS := $(patsubst plugins/%/scripts/gen_openapi_server.sh,%,$(wildcard plugins/*/scripts/gen_openapi_server.sh))

# Only removes files that gen_openapi_server.sh actually (re)creates for each
# plugin (api_<plugin>_catalog_service.go, api_<plugin>.go) plus shared
# generator output. Hand-written implementation files such as
# api_<plugin>_catalog_service_service.go (and their tests) must NOT match
# here, or `make clean gen` deletes manually maintained code.
.PHONY: clean-internal-server-openapi
clean-internal-server-openapi:
	for ver in v1alpha1 v1; do \
		for plugin in $(PLUGINS); do \
			rm -f internal/server/openapi/$$ver/api_$${plugin}_catalog_service.go internal/server/openapi/$$ver/api_$${plugin}.go; \
		done; \
		rm -f internal/server/openapi/$$ver/impl.go internal/server/openapi/$$ver/error.go internal/server/openapi/$$ver/helpers.go internal/server/openapi/$$ver/routers.go internal/server/openapi/$$ver/logger.go internal/server/openapi/$$ver/type_asserts.go internal/server/openapi/$$ver/model_*.go; \
	done

.PHONY: gen/catalog-plugin
gen/catalog-plugin:
	@test -n "$(NAME)" || (echo "Usage: make gen/catalog-plugin NAME=<plugin-name> DESCRIPTION=<desc> ENTITIES=<Entity:type,...>" && exit 1)
	@test -n "$(DESCRIPTION)" || (echo "Usage: make gen/catalog-plugin NAME=<plugin-name> DESCRIPTION=<desc> ENTITIES=<Entity:type,...>" && exit 1)
	@test -n "$(ENTITIES)" || (echo "Usage: make gen/catalog-plugin NAME=<plugin-name> DESCRIPTION=<desc> ENTITIES=<Entity:type,...>" && exit 1)
	$(GO) run ../tools/catalog-gen init --name=$(NAME) --description="$(DESCRIPTION)" $(foreach e,$(subst $(comma), ,$(ENTITIES)),--entity=$(e)) --root=..

comma := ,

.PHONY: test
test:
	${GO} test ./...

.PHONY: test-nocache
test-nocache:
	${GO} test ./... -count=1

.PHONY: test-cover
test-cover:
	${GO} test ./... -coverprofile=coverage.txt
	${GO} tool cover -html=coverage.txt -o coverage.html
