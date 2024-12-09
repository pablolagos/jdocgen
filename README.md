Here’s an updated `README.md` for your project:

---

# jdocgen

`jdocgen` is a Go-based tool for generating Markdown documentation for JSON-RPC 2.0 APIs. It parses your Go code to extract function annotations and generates comprehensive documentation, including parameter details, results, errors, and inline struct definitions.

## Installation

To install `jdocgen`, run:

```bash
go install github.com/pablolagos/jdocgen@latest
```

## Usage

Run `jdocgen` from your project directory to generate API documentation:

```bash
jdocgen
```

## Flags

| Flag          | Description                                      | Default                 |
|---------------|--------------------------------------------------|-------------------------|
| `-dir`        | Directory to parse for Go source files.          | `.` (current directory) |
| `-output`     | Path to the output Markdown file.                | `API_Documentation.md`  |
| `-omit-rfc`   | Omit JSON-RPC 2.0 specification from the output. | `false`                 |

---

## Project Annotations

Include project-level annotations in your source code comments to provide metadata about the API:

| Annotation     | Description                       | Example                                    |
|----------------|-----------------------------------|--------------------------------------------|
| `@Title`       | Title of the project.             | `@Title My API`                            |
| `@Version`     | Version of the project.           | `@Version 1.0.0`                           |
| `@Description` | Brief description of the project. | `@Description This API provides...`        |
| `@Author`      | Author of the project.            | `@Author John Doe`                         |
| `@License`     | License for the project.          | `@License MIT`                             |
| `@Contact`     | Contact information.              | `@Contact support@example.com`             |
| `@Terms`       | Link to terms and conditions.     | `@Terms https://example.com/terms`         |
| `@Repository`  | Repository URL for the project.   | `@Repository https://github.com/user/repo` |
| `@Tags`        | Tags associated with the project. | `@Tags jsonrpc, api, example`              |

---

## Function Annotations

Document your API endpoints by adding annotations to function comments:

| Annotation     | Description                                                                            | Example                                    |
|----------------|----------------------------------------------------------------------------------------|--------------------------------------------|
| `@Command`     | Command name for the JSON-RPC method.                                                  | `@Command stats.GetAllMetrics`             |
| `@Description` | Brief description of the endpoint.                                                     | `@Description Get statistics for 30 days.` |
| `@Parameter`   | Parameters accepted by the method. Format: `@Parameter <name> <type> "<description>"`. | `@Parameter tz string "Timezone."`         |
| `@Result`      | Return type and description. Format: `@Result <type> "<description>"`.                 | `@Result Stats "Statistics data."`         |
| `@Error`       | Errors returned by the method. Format: `@Error <code> "<description>"`.                | `@Error 400 "Invalid timezone."`           |
| `@Additional`  | Additional structs related to the endpoint. Format: `@Additional <struct>`             | `@Additional User`                         |

---

## Output Format

The generated Markdown includes:

1. **API Command Details**: Command name, description, parameters, results, and errors.
2. **JSON-RPC 2.0 Specification** (optional): Overview of the JSON-RPC protocol.
3. **Inline Struct Definitions**: Detailed documentation for all referenced structs.

Example output for a command:

```markdown
## stats.GetAllMetrics

Get statistics information for the last 30 days.

### Parameters:

| Name | Type   | Description | Required |
|------|--------|-------------|----------|
| tz   | string | Timezone.   | Yes      |

### Results:

| Name   | Type  | Description               |
|--------|-------|---------------------------|
| result | Stats | Statistics information.   |

#### Stats

| Name               | Type  | Description                     | JSON Name |
|--------------------|-------|---------------------------------|-----------|
| TotalScannedFiles  | []int | Total scanned files in 30 days. | total_scanned_files |
| TotalInfectedFiles | []int | Total infected files in 30 days.| total_infected_files |

---

### Additional Structs:

#### User

| Name         | Type    | Description | JSON Name |
|--------------|---------|-------------|-----------|
| UserName     | string  | User name.  | username  |
| Email        | string  | User email. | email     |
```

---

Feel free to adapt the examples and descriptions to your specific use case!