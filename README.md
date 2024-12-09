# jDocGen: JSON-RPC API Documentation Generator

**jDocGen** is a Go-based tool to generate Markdown documentation for JSON-RPC APIs. It parses Go source files to extract annotated functions and structures, creating a comprehensive API reference, including generic structs and their instantiations.

---

## Installation

To install `jDocGen`, run the following command:

```bash
go install github.com/pablolagos/jdocgen@latest
```

This will download and install the tool into your `$GOPATH/bin`.

---

## Usage

Run `jDocGen` from your project directory to generate the documentation:

```bash
jdocgen
```

### Flags

| Flag        | Default                | Description                                                                 |
|-------------|------------------------|-----------------------------------------------------------------------------|
| `-output`   | `API_Documentation.md`| Path to the output Markdown file.                                           |
| `-dir`      | `.`                    | Directory to parse for Go source files.                                     |
| `-omit-rfc` | `false`                | Omit the JSON-RPC 2.0 specification introduction in the documentation.      |

---

## How to Use Annotations

To document your API endpoints and related structures, use the following annotations in your code.

### 1. **Project-Level Tags**

Annotate your project in a Go file to provide metadata for the generated documentation.

Example:

```go
// @Project My API Documentation
// @Version 1.0.0
// @Author Jane Doe
// @Description This is the API documentation for My JSON-RPC service.
```

### 2. **Endpoint-Level Tags**

Use these tags in your handler functions to describe API endpoints:

| Tag          | Description                                                                                   |
|--------------|-----------------------------------------------------------------------------------------------|
| `@Command`   | The unique name of the command (method) in the JSON-RPC API.                                  |
| `@Description` | A short description of what the command does.                                                |
| `@Parameter` | Describes a parameter for the function in the format: `name type description [optional flag]`.|
| `@Result`    | Describes the return value of the function in the format: `type description`.                 |

Example:

```go
// GetAllMetrics returns statistics for the last 30 days.
//
// @Command stats.GetAllMetrics
// @Description Get statistics for the last 30 days.
// @Parameter tz string Timezone for the stats. [optional]
// @Result Stats Statistics information.
func (h *Handlers) GetAllMetrics(ctx *jrpc.Context) error {
	// Implementation here...
}
```

### 3. **Struct-Level Tags**

For structs, provide a description to include them in the documentation. Generic structs are supported and expanded when referenced by endpoints.

Example:

```go
// Pagination represents a paginated response.
// @description Pagination is a generic struct for paginated data.
type Pagination[T any] struct {
	Data       []T `json:"data"`        // Data for the current page
	Page       int `json:"page"`        // Current page number
	PageSize   int `json:"page_size"`   // Number of items per page
	TotalPages int `json:"total_pages"` // Total number of pages
	TotalItems int `json:"total_items"` // Total number of items available
}
```

---

## Example Output

### Endpoint Documentation

```markdown
## stats.GetAllMetrics

Get statistics for the last 30 days.

### Parameters:

| Name | Type   | Description              | Required |
|------|--------|--------------------------|----------|
| tz   | string | Timezone for the stats.  | Optional |

### Results:

| Name    | Type | Description           |
|---------|------|-----------------------|
| result  | Stats| Statistics information|

---

### Struct Documentation

#### Stats

| Name                | Type   | Description |
|---------------------|--------|-------------|
| TotalScannedFiles   | []int  | Total files scanned in the last 30 days. |
| TotalInfectedFiles  | []int  | Total infected files in the last 30 days.|
| QuarantinedFiles    | []int  | Files quarantined in the last 30 days.   |
```

---

## Features

- **Support for Generics**: Expands and documents generic structs with their instantiated types.
- **Detailed Endpoint Information**: Automatically includes parameters, results, and referenced structs.
- **Project Metadata**: Allows easy inclusion of project-level information such as version, author, and description.
- **JSON-RPC Specification**: Optionally includes an overview of JSON-RPC 2.0.

---

## Contributing

Contributions are welcome! If you encounter any issues or have suggestions, please [open an issue](https://github.com/pablolagos/jdocgen/issues) or submit a pull request.

---

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/pablolagos/jdocgen/blob/main/LICENSE) file for details.