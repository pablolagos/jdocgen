# jdocgen

![jdocgen Logo](https://github.com/pablolagosm/jdocgen/raw/main/logo.png) <!-- Optional: Add a logo if available -->

## Overview

**jdocgen** is a powerful CLI tool designed to automatically generate comprehensive Markdown documentation for your JSON-RPC APIs written in Go. By parsing specially annotated Go source files, `jdocgen` extracts essential information about your API functions, parameters, return values, and data structures, presenting them in a clean and organized Markdown format. This ensures that your API documentation stays up-to-date with your codebase, enhancing maintainability and developer experience.

## Features

- **Automatic Documentation Generation:** Extracts API metadata from annotated Go source files.
- **Explicit Optional Fields:** Clearly distinguishes between required and optional parameters and results.
- **Inline Struct Definitions:** Includes detailed descriptions of data structures used in API functions.
- **Global Project Metadata:** Incorporates project-wide information such as title, version, description, and more.
- **Easy Installation:** Installable via `go install`, making it accessible across different environments.
- **Customizable Output:** Specify output file names and target directories effortlessly.
- **Extensible Architecture:** Designed for easy integration of additional features and customization.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Annotation Syntax](#annotation-syntax)
  - [Global Tags](#global-tags)
  - [API Function Annotations](#api-function-annotations)
    - [Command Annotation](#command-annotation)
    - [Description Annotation](#description-annotation)
    - [Parameter Annotation](#parameter-annotation)
    - [Result Annotation](#result-annotation)
- [Example](#example)
  - [Annotated Source Files](#annotated-source-files)
  - [Generated Documentation](#generated-documentation)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

## Installation

To install `jdocgen`, ensure you have [Go](https://golang.org/dl/) installed on your system (version 1.18 or later).

Run the following command:

```bash
go install github.com/pablolagosm/jdocgen/cmd/jdocgen@latest
```

This will compile the `jdocgen` binary and place it in your `$GOPATH/bin` directory. Make sure this directory is included in your system's `PATH` to run `jdocgen` from anywhere.

**Verify Installation:**

```bash
jdocgen --help
```

**Expected Output:**

```
Usage of jdocgen:
  -dir string
        Root directory to parse for Go files (default ".")
  -output string
        Output file for the documentation (default "API_Documentation.md")
```

## Usage

After installation and annotating your Go source files, navigate to your project's root directory and run:

```bash
jdocgen -output=API_Documentation.md -dir=./
```

**Flags:**

- `-output`: Specifies the name of the output Markdown file (default: `API_Documentation.md`).
- `-dir`: Specifies the root directory of the Go project to parse (default: current directory).

**Example Command:**

```bash
jdocgen -output=MyAPI_Docs.md -dir=./
```

**Output:**

```
Documentation successfully generated at /absolute/path/to/your/project/MyAPI_Docs.md
```

## Annotation Syntax

To enable `jdocgen` to parse your Go source files effectively, you need to annotate your code using structured comments. These annotations provide metadata about your API functions, parameters, return values, and global project information.

### Global Tags

Global tags provide metadata about the entire project. These should be placed in the comments associated with the `main` function or at the very top of `main.go`.

**Mandatory Global Tags:**

- `@title`: The title of your API documentation.
- `@version`: The current version of your API.
- `@description`: A brief description of your API.

**Optional Global Tags:**

- `@author`
- `@license`
- `@contact`
- `@terms`
- `@repository`
- `@tags`
- `@copyright`

**Example:**

```go
// cmd/jdocgen/main.go

// @title JSON-RPC API Documentation
// @version 1.0.0
// @description This project provides a JSON-RPC API for managing users and products.
// @license Apache-2.0
// @contact api-support@example.com
// @terms https://example.com/terms
// @repository https://github.com/pablolagosm/jdocgen
// @tags User, Product, Management
// @copyright 
// © 2024 Your Company

package main

func main() {
	// Implementation here
}
```

### API Function Annotations

Each API function should be annotated with specific tags that describe its purpose, parameters, and return values.

#### Command Annotation

- **Tag:** `@Command`
- **Purpose:** Specifies the name of the API command.
- **Format:**

  ```
  // @Command <CommandName>
  ```

- **Example:**

  ```go
  // @Command CreateUser
  ```

#### Description Annotation

- **Tag:** `@Description`
- **Purpose:** Provides a brief description of what the API function does.
- **Format:**

  ```
  // @Description <Description>
  ```

- **Example:**

  ```go
  // @Description Creates a new user in the system.
  ```

#### Parameter Annotation

- **Tag:** `@Parameter`
- **Purpose:** Describes a parameter accepted by the API function.
- **Format:**

  ```
  // @Parameter <Name> <Type> [optional] <Description>
  ```

- **Notes:**
    - The keyword `optional` is case-insensitive and denotes that the parameter is not required.
    - If `optional` is present, the `Required` flag is set to `false`; otherwise, it's `true`.

- **Examples:**

  ```go
  // @Parameter user models.User The user data to create.
  // @Parameter updates *models.User optional The updated user data.
  ```

#### Result Annotation

- **Tag:** `@Result`
- **Purpose:** Describes a return value produced by the API function.
- **Format:**

  ```
  // @Result <Name> <Type> [optional] <Description>
  ```

- **Notes:**
    - The keyword `optional` is case-insensitive and denotes that the result is not guaranteed.
    - If `optional` is present, the `Required` flag is set to `false`; otherwise, it's `true`.

- **Examples:**

  ```go
  // @Result id string The unique identifier of the created user.
  // @Result err error Error in case of failure.
  // @Result profile *Profile optional Detailed profile information.
  ```

## Example

### Annotated Source Files

#### `cmd/jdocgen/main.go`

```go
// cmd/jdocgen/main.go

// @title JSON-RPC API Documentation
// @version 1.0.0
// @description This project provides a JSON-RPC API for managing users and products.
// @license Apache-2.0
// @contact api-support@example.com
// @terms https://example.com/terms
// @repository https://github.com/pablolagosm/jdocgen
// @tags User, Product, Management
// @copyright 
// © 2024 Your Company

package main

func main() {
	// Implementation here
}
```

#### `api/user.go`

```go
// api/user.go
package api

import "github.com/pablolagosm/jdocgen/models"

// @Command CreateUser
// @Description Creates a new user in the system.
// @Parameter user models.User The user data to create.
// @Result id string The unique identifier of the created user.
// @Result err error Error in case of failure.
func CreateUser(user models.User) (id string, err error) {
	// Implementation here
	return "unique-id", nil
}

// @Command UpdateUser
// @Description Updates an existing user's information.
// @Parameter id string The unique identifier of the user to update.
// @Parameter updates *models.User optional The updated user data.
// @Result success bool Indicates if the update was successful.
// @Result err error Error in case of failure.
func UpdateUser(id string, updates *models.User) (success bool, err error) {
	// Implementation here
	return true, nil
}

// @Command DeleteUser
// @Description Deletes a user from the system.
// @Parameter id string The unique identifier of the user to delete.
// @Result success bool Indicates if the deletion was successful.
// @Result err error Error in case of failure.
func DeleteUser(id string) (success bool, err error) {
	// Implementation here
	return true, nil
}
```

#### `models/user.go`

```go
// models/user.go
package models

// User represents a user in the system.
type User struct {
	// Username is the unique username of the user.
	Username string `json:"username"`
	// Email is the user's email address.
	Email string `json:"email"`
	// Age is the user's age.
	Age int `json:"age"`
}
```

### Generated Documentation

After running `jdocgen`, the `API_Documentation.md` will look like this:

```markdown
# JSON-RPC API Documentation

**Version:** 1.0.0

**Description:** This project provides a JSON-RPC API for managing users and products.

**Author:** John Doe

**License:** Apache-2.0

**Contact:** api-support@example.com

**Terms of Service:** https://example.com/terms

**Repository:** [https://github.com/pablolagosm/jdocgen](https://github.com/pablolagosm/jdocgen)

**Tags:** User, Product, Management

---

## API Overview

This document describes the functions available through the JSON-RPC API.

## `CreateUser`

Creates a new user in the system.

### Parameters

| Name | Type | Description | Required |
|------|------|-------------|----------|
| `user` | `models.User` | The user data to create. | Yes     |

#### `User` Structure

| Field    | Type   | Description                    |
|----------|--------|--------------------------------|
| `username` | `string` | Username is the unique username of the user. |
| `email`    | `string` | Email is the user's email address. |
| `age`      | `int`    | Age is the user's age. |

### Return Values

| Name | Type | Description | Required |
|------|------|-------------|----------|
| `id`   | `string` | The unique identifier of the created user. | Yes     |
| `err`  | `error` | Error in case of failure. | Yes     |

---

## `UpdateUser`

Updates an existing user's information.

### Parameters

| Name | Type | Description | Required |
|------|------|-------------|----------|
| `id` | `string` | The unique identifier of the user to update. | Yes     |
| `updates` | `*models.User` | The updated user data. | *No*     |

#### `User` Structure

| Field    | Type   | Description                    |
|----------|--------|--------------------------------|
| `username` | `string` | Username is the unique username of the user. |
| `email`    | `string` | Email is the user's email address. |
| `age`      | `int`    | Age is the user's age. |

### Return Values

| Name | Type | Description | Required |
|------|------|-------------|----------|
| `success` | `bool` | Indicates if the update was successful. | Yes     |
| `err`  | `error` | Error in case of failure. | Yes     |

---

## `DeleteUser`

Deletes a user from the system.

### Parameters

| Name | Type | Description | Required |
|------|------|-------------|----------|
| `id` | `string` | The unique identifier of the user to delete. | Yes     |

### Return Values

| Name | Type | Description | Required |
|------|------|-------------|----------|
| `success` | `bool` | Indicates if the deletion was successful. | Yes     |
| `err`  | `error` | Error in case of failure. | Yes     |

---
```

**Highlights:**

- **Clear Indicators for Optional Fields:** The `updates` parameter in the `UpdateUser` function is marked as `*No*` in the "Required" column, indicating its optionality.
- **Clean Descriptions:** The `optional` keyword does not appear in the descriptions, maintaining readability.
- **Consistent Structure:** Each API function follows a consistent format, enhancing the overall coherence of the documentation.

## Contributing

Contributions are welcome! Whether it's reporting bugs, suggesting features, or improving documentation, your input helps make `jdocgen` better.

1. **Fork the Repository**

   Click the [Fork](https://github.com/pablolagosm/jdocgen/fork) button at the top right of this page.

2. **Clone Your Fork**

   ```bash
   git clone https://github.com/yourusername/jdocgen.git
   ```

3. **Create a Branch**

   ```bash
   git checkout -b feature/YourFeature
   ```

4. **Make Your Changes**

   Implement your feature or bug fix.

5. **Commit Your Changes**

   ```bash
   git commit -m "Add feature: YourFeature"
   ```

6. **Push to Your Fork**

   ```bash
   git push origin feature/YourFeature
   ```

7. **Open a Pull Request**

   Navigate to the original repository and open a pull request detailing your changes.

## License

This project is licensed under the MIT License.
