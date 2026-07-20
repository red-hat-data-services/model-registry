---
name: create-agent-catalog-source
description: >
  Use this skill when the user asks to "create an agent catalog", "generate agent catalog source",
  "build agent catalog from repo", "catalog my agents", or wants to scan a repository containing
  agent templates and/or agent metadata to produce a custom agent catalog source YAML for the
  agents catalog. Trigger phrases include: "create agent catalog source",
  "generate agent catalog", "catalog agents from repo", "agent catalog from repository",
  "custom agent catalog".
allowed-tools: Bash Read Grep Glob Write AskUserQuestion WebFetch
---

# Create Agent Catalog Source

Scan a repository for agent templates and metadata, then generate a custom agent catalog source
compatible with the agents catalog service.

**Usage:**
- `/create-agent-catalog-source` — scan the current working directory
- `/create-agent-catalog-source /path/to/repo` — scan a local repository
- `/create-agent-catalog-source https://github.com/org/repo` — clone and scan a remote repository

---

## Step 1: Resolve Input

Parse the argument passed by the user to determine the target repository path.

**If the argument looks like a GitHub URL** (starts with `https://github.com/` or `git@github.com:`):
1. Generate a temporary directory path:
   ```bash
   TMPDIR=$(mktemp -d /tmp/agent-catalog-XXXXXX)
   ```
2. Clone the repository (shallow):
   ```bash
   git clone --depth 1 <url> "$TMPDIR"
   ```
3. If the clone fails, report the error and suggest the user clone manually and pass the local path instead. Stop.
4. Use `$TMPDIR` as the scan target. Remember to clean up at the end.

**If the argument is a local path:**
1. Verify the path exists:
   ```bash
   test -d <path>
   ```
2. If it does not exist, report the error and stop.
3. Use the provided path as the scan target.

**If no argument is provided:**
1. Use the current working directory as the scan target.

**After resolving the target path**, detect git metadata for `repositoryUrl` generation:
```bash
cd <target-path>
git remote get-url origin 2>/dev/null
git branch --show-current 2>/dev/null
```
If the directory is not a git repo, warn the user that `repositoryUrl` fields will be empty but continue.

Convert the git remote URL to an HTTPS base URL:
- `git@github.com:org/repo.git` → `https://github.com/org/repo`
- `https://github.com/org/repo.git` → `https://github.com/org/repo`
- Store the remote URL and branch name for later use when building `repositoryUrl` per agent.

**Check the output directory:**
```bash
test -d <original-cwd>/.agent-catalog
```
If `.agent-catalog/` already exists, ask the user whether to overwrite or choose a different output
path using `AskUserQuestion`. Store the chosen path as `<output-dir>` (default: `<original-cwd>/.agent-catalog`).
All subsequent steps use `<output-dir>` when writing output files.

---

## Step 2: Discover Agents — Convention Scan

Search the target repository for directories containing an `agent.yaml` file. This is the
preferred agent metadata format. Inform the user:

> "Scanning for agent.yaml files (preferred convention)..."

**Find all agent.yaml files:**
```bash
find <target-path> -name "agent.yaml" \
  -not -path "*/.git/*" \
  -not -path "*/.github/*" \
  -not -path "*/node_modules/*" \
  -not -path "*/vendor/*" \
  -not -path "*/build/*" \
  -not -path "*/dist/*" \
  -not -path "*/__pycache__/*"
```

**For each `agent.yaml` found:**

1. Read the file content.
2. Extract known fields:
   - `name` (string, required)
   - `displayName` (string, optional)
   - `framework` (string, required)
   - `description` (string, required)
   - `labels` (string array, optional, default `[]`)
   - `logo` (string, optional, default `""`)
   - `env` (object with `required` and `optional` string arrays, optional)
3. Note any additional top-level fields — these will become `customProperties`.
4. Check for a companion README.md in the same directory:
   ```bash
   test -f <agent-dir>/README.md
   ```
5. If README.md exists, read its full content for the `readme` field.
6. Compute the agent's relative path from the repo root (for `repositoryUrl`).

**Collect all discovered agents into a list.** Track for each:
- Relative path within the repo
- Parsed `agent.yaml` fields
- Whether README.md was found
- Any extra fields for customProperties
- Discovery source: `"convention"`

**Validation per agent:**
- If `name` is missing, derive it from the directory name (kebab-case).
- If `framework` is missing, flag it as needing user input.
- If `description` is missing, try to extract the first paragraph from README.md if available.

**Always proceed to Step 3 (Heuristic Scan)** regardless of how many agents the convention
scan found. The heuristic scan may discover additional agents in non-standard formats
(e.g., YAML files not named `agent.yaml`, or agents documented only in Markdown).

---

## Step 3: Discover Agents — Heuristic Scan

This step always runs after the convention scan to find additional agents that may use
non-standard file names or only have Markdown documentation.

Inform the user:
> "Scanning for additional agents via YAML heuristic and Markdown analysis..."

### 3a: YAML Heuristic Scan

Search for YAML files that look like they describe an agent:

```bash
find <target-path> -type f \( -name "*.yaml" -o -name "*.yml" \) \
  -not -path "*/.git/*" \
  -not -path "*/node_modules/*" \
  -not -path "*/vendor/*" \
  -not -name "docker-compose*" \
  -not -name "*lock*" \
  -not -name ".pre-commit*"
```

**For each YAML file**, read it and check if it is "agent-like":
- It must contain a `name` field (string)
- It must contain at least one of: `framework`, `description`, or `env`
- Exclude files that look like CI configs, Dockerfiles, Kubernetes manifests (check for
  `apiVersion`, `kind`, `services`, `jobs`, `steps` top-level keys — these are not agents)

For qualifying files:
- Extract the same known fields as the convention scan
- Set discovery source to `"yaml-heuristic"`
- Note the file path

### 3b: Markdown Scan

Search for Markdown files that describe agents:

```bash
find <target-path> -type f -name "*.md" \
  -not -path "*/.git/*" \
  -not -path "*/node_modules/*" \
  -not -name "CHANGELOG*" \
  -not -name "LICENSE*" \
  -not -name "CONTRIBUTING*"
```

**For each Markdown file**, scan for agent-relevant signals:

1. **Framework mentions** — search (case-insensitive) for:
   `langgraph`, `crewai`, `autogen`, `llamaindex`, `llama-index`, `haystack`,
   `semantic-kernel`, `langflow`, `langchain`, `openai-agents`, `google-adk`, `a2a`

2. **Agent keywords** — search for:
   `agent`, `tool-calling`, `ReAct`, `chain-of-thought`, `orchestrator`, `agentic`,
   `function calling`, `tool use`

3. **Environment variable documentation** — look for sections with headers like
   "Environment Variables", "Configuration", "Setup", ".env" that list variable names

4. **Description extraction** — extract the first paragraph after the top-level heading
   as a candidate description

A Markdown file is a candidate if it has at least one framework mention AND at least one
agent keyword.

For qualifying Markdown files, extract as many of the **7 required catalog fields** as possible:
- `name` — infer from the filename or first heading (kebab-cased)
- `displayName` — infer from the first heading (original casing)
- `framework` — from the framework mention found
- `description` — from the first paragraph after the top-level heading
- `labels` — default `[]` (user can add during confirmation)
- `logo` — default `""` (user can provide during confirmation)
- `env` — **extract from environment variable documentation**: look for bullet lists or
  tables listing variable names. Parse `required`/`optional` indicators from the text
  (e.g., "required", "optional", "Yes", "No"). If a var is listed without an indicator,
  default to `required: true`
- Set `readme` to the full file content
- Set discovery source to `"markdown-inference"`
- Flag all fields as **draft** — the user must confirm/edit

### Merge Results

Combine convention-scan, YAML-heuristic, and markdown-inference results into a single list:

1. **Deduplicate:** If a heuristic/markdown candidate is from the same directory as a
   convention-scan agent, skip the heuristic candidate (convention already captured it).
2. **Merge within heuristic:** If a YAML-heuristic candidate and a markdown-inference
   candidate are from the same directory, merge them — prefer YAML fields, supplement
   with Markdown-extracted description/readme.

**Proceed to Step 3c (Generate agent.yaml) with the merged list.**

If zero agents total after all scans, tell the user:

> "No agents found in this repository. To use this skill, organize your agents with an
> `agent.yaml` file per agent directory. Here is the expected format:"

Then print an example `agent.yaml`:
```yaml
name: my-agent
displayName: My Agent
framework: langgraph
description: A custom agent that does X
labels:
  - tool-calling
  - rag
env:
  required:
    - API_KEY
    - MODEL_ENDPOINT
  optional:
    - DEBUG
```

And stop.

### 3c: Generate agent.yaml for Heuristic-Discovered Agents

Every agent in the catalog must have an `agent.yaml` file. Convention-scanned agents already
have one. For each agent discovered via `yaml-heuristic` or `markdown-inference`, generate
an `agent.yaml` matching the format expected by the agents catalog.

**Important:** The generated `agent.yaml` is NOT written to the agent's source directory —
it is assembled in memory and included only in the catalog output (as the `templates` entry
in `<output-dir>/catalog.yaml`). The agent's source directory remains unchanged.

**For each heuristic/markdown-discovered agent:**

1. **Assemble the agent.yaml content** from the extracted metadata using this exact format:

   ```yaml
   name: <kebab-case-name>
   displayName: "<Human Readable Display Name>"
   framework: <framework>
   description: "<description>"
   labels: [<labels-as-inline-array-or-empty>]
   logo: "<logo-string-or-empty>"
   env:
     required:
       - VAR_NAME_1
       - VAR_NAME_2
     optional:
       - VAR_NAME_3
   ```

   Field mapping:
   - `name`: from extracted metadata, kebab-cased
   - `displayName`: from extracted metadata, or title-cased from name if not available
   - `framework`: from extracted metadata
   - `description`: from extracted metadata
   - `labels`: from extracted metadata, default `[]`
   - `logo`: from extracted metadata, default `""`
   - `env.required` / `env.optional`: from extracted environment variable documentation.
     If no env vars were found, use `required: []` and `optional: []`

2. **Display the generated agent.yaml** to the user:
   > "Generated `agent.yaml` for agent `<name>` (discovered via `<source>`):"
   >
   > ```yaml
   > <full assembled content>
   > ```

3. **Use `AskUserQuestion`** to ask the user to confirm:
   - **"Accept (Recommended)"** — the generated agent.yaml is correct
   - **"Deny"** — reject this agent; it will be **excluded entirely** from the catalog

4. **If the user chooses Accept:**
   - Ask: "Would you like to amend any fields before finalizing? (y/N)"
   - If yes: let the user specify which fields to change and their new values.
     Update the assembled content. Display the updated version for final confirmation.
   - Store the confirmed agent.yaml content for use in Step 5 (templates field).

5. **If the user chooses Deny:**
   - Remove this agent from the discovered agents list.
   - Inform the user:
     > "Agent `<name>` has been excluded from the catalog."

6. **After processing all heuristic/markdown agents**, report the final count:
   > "N agents confirmed with agent.yaml files. Proceeding to catalog source configuration."

   If all heuristic/markdown agents were denied and no convention agents exist either,
   stop with the same "No agents found" message as above.

**Proceed to Step 4 with only the confirmed agents.**

---

## Step 4: User Confirmation

Present all discovered agents to the user in a summary table. Format:

```
Discovered N agents:

  #  Name                     Framework    Source           Path
  1. langgraph-react-agent    langgraph    convention       agents/react_agent/
  2. crewai-rag-agent         crewai       convention       agents/rag/
  3. my-custom-agent          autogen      yaml-heuristic   tools/agent.yaml
  4. search-bot               langgraph    markdown         bots/search/README.md
```

For `markdown-inference` sourced agents, also show the draft metadata:
```
  Agent #4 (search-bot) — inferred from Markdown, please review:
    name: search-bot
    framework: langgraph (inferred from README mention)
    description: "A search bot that uses LangGraph to orchestrate web search tools"
    [Edit any field? y/N]
```

**Use `AskUserQuestion` to ask:**

1. "Include all discovered agents, or select which to include?"
   - Options: "Include all (Recommended)", "Let me select", "Add agents manually"
   - If "Let me select": ask which numbers to include (comma-separated)
   - If "Add agents manually": ask for the path to an agent directory, read its files,
     and add it to the list. Repeat until the user is done.

2. **Required fields check.** Every agent in the catalog MUST have all 7 of these fields
   (even if some use default/empty values):
   - `name` (string, non-empty)
   - `displayName` (string, non-empty)
   - `description` (string, non-empty)
   - `framework` (string, non-empty)
   - `labels` (string array, may be `[]`)
   - `logo` (string, may be `""`)
   - `env` (array of `{name, required}`, may be `[]`)

   For any agent missing a non-empty required field (`name`, `displayName`, `description`,
   or `framework`), ask the user to provide the value. For `labels`, `logo`, and `env`,
   use defaults (`[]`, `""`, `[]`) if not available, but show the user what defaults were
   applied so they can override.

3. For `markdown-inference` agents, present each one's draft metadata for ALL 7 required
   fields and ask if the user wants to edit any. Let them type corrections inline.

4. "What should the catalog source be named?"
   - Default: the repository name (derived from git remote or directory name)
   - The user can accept or type a custom name

5. "Would you like to add labels to this catalog source? (e.g., 'Custom', 'Internal')"
   - Default: no labels (empty array)
   - If yes, ask for comma-separated label strings

Store the confirmed agents list, source name, and source labels for output generation.

---

## Step 5: Generate Catalog YAML

Create the output directory:
```bash
mkdir -p <output-dir>
```

**Build the catalog YAML content.** The output must match this exact schema (compatible with
the model-registry `yamlAgentCatalog` consumer):

```yaml
source: <confirmed-source-name>
agents:
    - name: <kebab-case-name>
      displayName: <display-name>
      description: <description>
      readme: |
        <full README.md content, or empty string>
      repositoryUrl: <git-remote-https-url>/tree/<branch>/<agent-relative-path>
      framework: <framework>
      labels:
        - <label1>
        - <label2>
      logo: "<logo-string-or-empty>"
      env:
        - name: <var-name>
          required: true
        - name: <var-name>
          required: false
      templates:
        - name: agent.yaml
          content: '<full-agent-yaml-as-json-string>'
      customProperties:
        <extra-field-name>:
            metadataType: MetadataStringValue
            string_value: "<value>"
```

**Field assembly per agent:**

| Field | How to build |
|-------|-------------|
| `name` | From confirmed metadata. Must be kebab-case and unique within the catalog. |
| `displayName` | From confirmed metadata. If empty, title-case the name (replace hyphens with spaces, capitalize each word). |
| `description` | From confirmed metadata. |
| `readme` | Full content of the agent's README.md. Use YAML literal block scalar (`|`). Empty string if no README. |
| `repositoryUrl` | `<git-remote-url>/tree/<branch>/<agent-relative-path>`. Empty string if no git remote was detected. |
| `framework` | From confirmed metadata. |
| `labels` | From agent.yaml `labels` field. Default `[]`. **Always include this field.** |
| `logo` | From agent.yaml `logo` field. Default `""`. **Always include this field.** |
| `env` | Transform from `{required: [...], optional: [...]}` to flat list: each required var gets `{name: X, required: true}`, each optional var gets `{name: X, required: false}`. For markdown-inferred agents, use env vars extracted from the README. Default `[]`. **Always include this field.** |
| `templates` | The `templates` content must always reflect the **final confirmed metadata** — including any edits the user made in Step 4. Convert the final metadata to a compact JSON string using `./bin/yq`. The result becomes a single template entry: `{name: "agent.yaml", content: "<json>"}`. **For all agents**, assemble a YAML representation of the final confirmed fields, write it to a temporary file (`TMPFILE=$(mktemp /tmp/agent-yaml-XXXXXX.yaml)`), run `./bin/yq "$TMPFILE" -o json -I 0`, capture stdout, then clean up (`rm "$TMPFILE"`). **For convention-discovered agents** where the user edited fields in Step 4: the on-disk `agent.yaml` is now out of sync — ask the user whether they want to update the original `agent.yaml` file in the source repository to match. **Always include this field.** |
| `customProperties` | For each top-level field in agent.yaml NOT in the known set (`name`, `displayName`, `framework`, `description`, `labels`, `logo`, `env`), create an entry: `{metadataType: "MetadataStringValue", string_value: "<value-as-string>"}`. Omit the field entirely if there are no extra fields. |

**IMPORTANT:** The 7 required fields (`name`, `displayName`, `description`, `framework`,
`labels`, `logo`, `env`) must be present on EVERY agent entry in the catalog, regardless
of discovery source. Use empty defaults (`[]`, `""`) where no value was found.

**Write the catalog file** to `<output-dir>/catalog.yaml`.

Ensure the file ends with a newline character.

**Validate:** Read back the written file and confirm:
- It is valid YAML
- It has a `source:` key
- It has an `agents:` array with the expected number of entries
- Each agent has ALL 7 required fields present: `name`, `displayName`, `description`,
  `framework`, `labels`, `logo`, `env`

Report to the user:

> "Catalog written to `<output-dir>/catalog.yaml` with N agents."

---

## Step 6: Generate Deployment Artifacts

### 6a: Sources Config Snippet

Derive the source ID from the confirmed source name:
- Lowercase the name
- Replace spaces with underscores
- Remove any characters that are not alphanumeric or underscores

Write `<output-dir>/sources-snippet.yaml` with the following content:

```yaml
- name: "<confirmed-source-name>"
  id: <derived-source-id>
  type: yaml
  enabled: true
  properties:
      yamlCatalogPath: /data/custom-agents/catalog.yaml
  labels:
    - <label1>
```

The `yamlCatalogPath` uses a placeholder — the user will update this to match the actual
mount path in their deployment.

If the user chose no labels, use: `labels: []`

### 6b: Deploy Script

Write `<output-dir>/deploy.sh` with the following content:

```bash
#!/bin/bash
# Deploy custom agent catalog source to Kubernetes
# Generated by /create-agent-catalog-source
#
# Prerequisites:
#   - kubectl (or oc) CLI logged into the target cluster
#   - Appropriate permissions to create ConfigMaps in the target namespace
#
# Usage:
#   cd <project-root>
#   NAMESPACE=my-namespace <output-dir>/deploy.sh

set -euo pipefail

NAMESPACE="${NAMESPACE:-kubeflow}"
CONFIGMAP_NAME="<derived-source-id>-catalog"

echo "Creating ConfigMap '$CONFIGMAP_NAME' in namespace '$NAMESPACE'..."

kubectl create configmap "$CONFIGMAP_NAME" \
    --from-file=catalog.yaml=<output-dir>/catalog.yaml \
    -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo ""
echo "ConfigMap created successfully."
echo ""
echo "Next steps:"
echo "  1. Mount the ConfigMap into the model-registry catalog deployment."
echo "  2. Edit the existing catalog sources ConfigMap to add this agent source:"
echo "     kubectl edit configmap <sources-configmap-name> -n $NAMESPACE"
echo ""
echo "  Add the following under the 'agent_catalogs' section:"
echo ""
cat <output-dir>/sources-snippet.yaml
echo ""
echo "  3. Update the yamlCatalogPath to match the mount path in your deployment."
echo "  4. The catalog service will hot-reload the new source automatically."
```

Make the deploy script executable:
```bash
chmod +x <output-dir>/deploy.sh
```

Replace `<derived-source-id>` in the script with the actual derived source ID value
before writing.

### 6c: Cleanup

If a temporary clone was created in Step 1:
```bash
rm -rf "$TMPDIR"
```

### 6d: Final Summary

Print to the user:

```
Agent catalog source generated successfully!

Output directory: <output-dir>/
  catalog.yaml          — Agent catalog data (N agents)
  sources-snippet.yaml  — Sources config snippet (paste into existing sources ConfigMap)
  deploy.sh             — Deployment script for Kubernetes

To deploy:
  1. Run: <output-dir>/deploy.sh
  2. Edit the existing sources ConfigMap to add the snippet from sources-snippet.yaml
  3. The catalog service will hot-reload automatically — no restart needed.

Source name: <confirmed-source-name>
Source ID:   <derived-source-id>

Note: Source IDs must be unique across all catalog types (models, MCP servers, agents)
in your model-registry deployment. If you have an existing source with ID
'<derived-source-id>', choose a different name.
```
