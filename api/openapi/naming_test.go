package openapi

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// This test enforces the parameter and field naming convention for the v1 APIs, as described in
// proposals/KEP-0004-api-v1-graduation/README.md:
//
//   - path parameters:            snake_case
//   - query parameters:           camelCase
//   - body/response fields:       camelCase
//
// The alpha spec started with this same convention but drifted over time because nothing
// enforced it. This test exists so that drift in the v1 spec is caught in CI instead.
//
// A small set of fields are intentionally exempt from the body/response convention — see
// fieldNamingExceptions below.
var (
	camelCaseRE = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	snakeCaseRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	httpMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}
)

// fieldNamingExceptions lists schema properties that are intentionally exempt from the
// camelCase convention, keyed by the schema they belong to. These mirror the MLMD/protobuf
// MetadataValue oneOf union field names and must keep their proto casing for wire
// compatibility with the underlying metadata store.
var fieldNamingExceptions = map[string]map[string]bool{
	"MetadataBoolValue":   {"bool_value": true},
	"MetadataDoubleValue": {"double_value": true},
	"MetadataIntValue":    {"int_value": true},
	"MetadataProtoValue":  {"proto_value": true},
	"MetadataStringValue": {"string_value": true},
	"MetadataStructValue": {"struct_value": true},
}

func TestV1SpecNamingConventions(t *testing.T) {
	specs := []string{
		"model-registry-v1.yaml",
		"catalog-v1.yaml",
	}

	for _, spec := range specs {
		t.Run(spec, func(t *testing.T) {
			data, err := os.ReadFile(spec)
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(data, &doc))

			var violations []string
			violations = append(violations, checkParameterNaming(doc)...)
			violations = append(violations, checkPropertyNaming(doc)...)

			violations = dedupe(violations)
			sort.Strings(violations)
			if len(violations) > 0 {
				t.Errorf("%s has %d naming convention violation(s):\n%s", spec, len(violations), strings.Join(violations, "\n"))
			}
		})
	}
}

// checkParameterNaming walks every path item and operation, resolving both inline parameters
// and $ref'd components/parameters entries, and verifies query parameters are camelCase and
// path parameters are snake_case.
func checkParameterNaming(doc map[string]any) []string {
	var violations []string

	componentParams, _ := digMap(doc, "components", "parameters")

	resolve := func(raw any) map[string]any {
		p, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		if ref, ok := p["$ref"].(string); ok {
			key := lastSegment(ref)
			if resolved, ok := componentParams[key].(map[string]any); ok {
				return resolved
			}
			return nil
		}
		return p
	}

	check := func(location string, rawParams any) {
		list, ok := rawParams.([]any)
		if !ok {
			return
		}
		for _, rawParam := range list {
			param := resolve(rawParam)
			if param == nil {
				continue
			}
			name, _ := param["name"].(string)
			in, _ := param["in"].(string)
			if name == "" {
				continue
			}

			switch in {
			case "query":
				if !camelCaseRE.MatchString(name) {
					violations = append(violations, fmt.Sprintf("%s: query parameter %q must be camelCase", location, name))
				}
			case "path":
				// RFC 6570 reserved expansion, e.g. {model_name+}, has a trailing '+' that
				// isn't part of the parameter name.
				trimmed := strings.TrimSuffix(name, "+")
				if !snakeCaseRE.MatchString(trimmed) {
					violations = append(violations, fmt.Sprintf("%s: path parameter %q must be snake_case", location, name))
				}
			}
		}
	}

	paths, _ := doc["paths"].(map[string]any)
	for path, rawItem := range paths {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		check(path, item["parameters"])
		for _, method := range httpMethods {
			op, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			check(fmt.Sprintf("%s %s", strings.ToUpper(method), path), op["parameters"])
		}
	}

	return violations
}

// checkPropertyNaming recursively collects every schema "properties" key reachable from
// components/schemas and from operation requestBody and response body schemas (inline or
// referenced), and verifies each one is camelCase.
func checkPropertyNaming(doc map[string]any) []string {
	var violations []string
	visited := map[string]bool{}

	schemas, _ := digMap(doc, "components", "schemas")

	var walk func(location string, node any)
	walk = func(location string, node any) {
		m, ok := node.(map[string]any)
		if !ok {
			return
		}

		if ref, ok := m["$ref"].(string); ok {
			key := lastSegment(ref)
			if visited[key] {
				return
			}
			visited[key] = true
			if target, ok := schemas[key].(map[string]any); ok {
				walk(key, target)
			}
			return
		}

		if props, ok := m["properties"].(map[string]any); ok {
			for name, propSchema := range props {
				if !camelCaseRE.MatchString(name) && !fieldNamingExceptions[location][name] {
					violations = append(violations, fmt.Sprintf("%s: field %q must be camelCase", location, name))
				}
				walk(location+"."+name, propSchema)
			}
		}

		for _, key := range []string{"allOf", "oneOf", "anyOf"} {
			if members, ok := m[key].([]any); ok {
				for _, member := range members {
					walk(location, member)
				}
			}
		}

		if items, ok := m["items"]; ok {
			walk(location, items)
		}
		if additional, ok := m["additionalProperties"]; ok {
			walk(location, additional)
		}
	}

	// Every named schema, so unreferenced schemas are still checked.
	for name, schema := range schemas {
		walk(name, schema)
	}

	// Inline request body schemas (e.g. multipart bodies) that aren't registered as a named
	// component schema.
	paths, _ := doc["paths"].(map[string]any)
	for path, rawItem := range paths {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		for _, method := range httpMethods {
			op, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			content, ok := digMap(op, "requestBody", "content")
			if !ok {
				continue
			}
			for mediaType, rawMedia := range content {
				media, ok := rawMedia.(map[string]any)
				if !ok {
					continue
				}
				location := fmt.Sprintf("%s %s requestBody (%s)", strings.ToUpper(method), path, mediaType)
				walk(location, media["schema"])
			}
		}
	}

	// Inline response body schemas. Responses are keyed by status code (or "default"), each
	// potentially $ref'ing components/responses, so resolve that indirection before digging
	// into content.
	componentResponses, _ := digMap(doc, "components", "responses")
	for path, rawItem := range paths {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		for _, method := range httpMethods {
			op, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			responses, ok := op["responses"].(map[string]any)
			if !ok {
				continue
			}
			for status, rawResponse := range responses {
				response, ok := rawResponse.(map[string]any)
				if !ok {
					continue
				}
				if ref, ok := response["$ref"].(string); ok {
					response, ok = componentResponses[lastSegment(ref)].(map[string]any)
					if !ok {
						continue
					}
				}
				content, ok := response["content"].(map[string]any)
				if !ok {
					continue
				}
				for mediaType, rawMedia := range content {
					media, ok := rawMedia.(map[string]any)
					if !ok {
						continue
					}
					location := fmt.Sprintf("%s %s response %s (%s)", strings.ToUpper(method), path, status, mediaType)
					walk(location, media["schema"])
				}
			}
		}
	}

	return violations
}

// digMap descends into nested map[string]any values by key, returning (nil, false) if any
// segment along the path is missing or not a map.
func digMap(m map[string]any, keys ...string) (map[string]any, bool) {
	cur := m
	for _, k := range keys {
		next, ok := cur[k].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// dedupe removes duplicate violation strings. The same schema can be reached both directly
// (as a top-level components/schemas entry) and indirectly (via a $ref from an allOf/oneOf/
// anyOf member elsewhere), so the same violation can otherwise be reported more than once.
func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func lastSegment(ref string) string {
	idx := strings.LastIndex(ref, "/")
	if idx == -1 {
		return ref
	}
	return ref[idx+1:]
}
