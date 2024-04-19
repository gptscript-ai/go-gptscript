# go-gptscript

This module provides a set of functions to interact with gptscripts. It allows for executing scripts, listing available tools and models, and more.

## Installation

To use this module, you need to have Go installed on your system. Then, you can install the module via:

```bash
go get github.com/gptscript-ai/go-gptscript
```

## Usage

To use the module, you need to first set the OPENAI_API_KEY environment variable to your OpenAI API key.

Additionally, you need the `gptscript` binary. You can install it on your system using the [installation instructions](https://github.com/gptscript-ai/gptscript?tab=readme-ov-file#1-install-the-latest-release).

## Functions

### listTools

Lists all the available built-in tools.

**Usage:**

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func listTools(ctx context.Context) (string, error) {
	return gogptscript.ListTools(ctx)
}
```

### listModels

Lists all the available models, returns a list.

**Usage:**

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func listModels(ctx context.Context) ([]string, error) {
	return gogptscript.ListModels(ctx)
}
```

### ExecTool

Executes a prompt with optional arguments.

**Options:**

These are optional options that can be passed to the `ExecTool` functions.
Neither option is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching. Default (true).
- `cacheDir`: Specify the cache directory.

**Usage:**

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func runTool(ctx context.Context) (string, error) {
	t := gogptscript.Tool{
		Instructions: "who was the president of the united states in 1928?",
	}

	return gogptscript.ExecTool(ctx, gogptscript.Opts{}, t)
}
```

### ExecFile

Executes a GPT script file with optional input and arguments.

**Options:**

These are optional options that can be passed to the `ExecFile` function.
Neither option is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching.
- `cacheDir`: Specify the cache directory.

**Usage:**

The script is relative to the callers source directory.

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func execFile(ctx context.Context) (string, error) {
	opts := gogptscript.Opts{
		Cache: &[]bool{false}[0],
	}

	return gogptscript.ExecFile(ctx, "./hello.gpt", "--input World", opts)
}
```

### StreamExecTool

Executes a gptscript with optional input and arguments, and returns the output streams.

**Options:**

These are optional options that can be passed to the `StreamExecTool` function.
Neither option is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching.
- `cacheDir`: Specify the cache directory.

**Usage:**

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func streamExecTool(ctx context.Context) error {
	t := gogptscript.Tool{
		Instructions: "who was the president of the united states in 1928?",
	}

	stdOut, stdErr, wait := gogptscript.StreamExecTool(ctx, gogptscript.Opts{}, t)

	// Read from stdOut and stdErr before call wait()

	return wait()
}
```

### StreamExecToolWithEvents

Executes a gptscript with optional input and arguments, and returns the stdout, stderr, and gptscript events streams.

**Options:**

These are optional options that can be passed to the `StreamExecTool` function.
Neither option is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching.
- `cacheDir`: Specify the cache directory.

**Usage:**

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func streamExecTool(ctx context.Context) error {
	t := gogptscript.Tool{
		Instructions: "who was the president of the united states in 1928?",
	}

	stdOut, stdErr, events, wait := gogptscript.StreamExecToolWithEvents(ctx, gogptscript.Opts{}, t)

	// Read from stdOut and stdErr before call wait()

	return wait()
}
```

### streamExecFile

**Options:**

These are optional options that can be passed to the `exec` function.
Neither option is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching.
- `cacheDir`: Specify the cache directory.

**Usage:**

The script is relative to the callers source directory.

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func streamExecTool(ctx context.Context) error {
	opts := gogptscript.Opts{
		Cache: &[]bool{false}[0],
	}

	stdOut, stdErr, wait := gogptscript.StreamExecFile(ctx, "./hello.gpt", "--input world", opts)

	// Read from stdOut and stdErr before call wait()

	return wait()
}
```

### streamExecFileWithEvents

**Options:**

These are optional options that can be passed to the `exec` function.
Neither option is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching.
- `cacheDir`: Specify the cache directory.

**Usage:**

The script is relative to the callers source directory.

```go
package main

import (
	"context"

	gogptscript "github.com/gptscript-ai/go-gptscript"
)

func streamExecTool(ctx context.Context) error {
	opts := gogptscript.Opts{
		Cache: &[]bool{false}[0],
	}

	stdOut, stdErr, events, wait := gogptscript.StreamExecFileWithEvents(ctx, "./hello.gpt", "--input world", opts)

	// Read from stdOut and stdErr before call wait()

	return wait()
}
```

## Types

### Tool Parameters

| Argument          | Type           | Default     | Description                                                                                   |
|-------------------|----------------|-------------|-----------------------------------------------------------------------------------------------|
| name              | string         | `""`        | The name of the tool. Optional only on the first tool if there are multiple tools defined.                                                                         |
| description       | string         | `""`        | A brief description of what the tool does, this is important for explaining to the LLM when it should be used.                                                    |
| tools             | array          | `[]`        | An array of tools that the current tool might depend on or use.                               |
| maxTokens         | number/undefined | `undefined` | The maximum number of tokens to be used. Prefer `undefined` for uninitialized or optional values. |
| model             | string         | `""`        | The model that the tool uses, if applicable.                                                  |
| cache             | boolean        | `true`      | Whether caching is enabled for the tool.                                                      |
| temperature       | number/undefined | `undefined` | The temperature setting for the model, affecting randomness. `undefined` for default behavior. |
| args              | object         | `{}`        | Additional arguments specific to the tool, described by key-value pairs.                      |
| internalPrompt    | boolean  | `false`        | An internal prompt used by the tool, if any.                                                  |
| instructions      | string         | `""`        | Instructions on how to use the tool.                                                          |
| jsonResponse      | boolean        | `false`     | Whether the tool returns a JSON response instead of plain text. You must include the word 'json' in the body of the prompt                               |

### FreeForm Parameters

| Argument  | Type   | Default | Description                           |
|-----------|--------|---------|---------------------------------------|
| content   | string | `""`    | This is a multi-line string that contains the  entire contents of a valid gptscript file|

## License

Copyright (c) 2024, [Acorn Labs, Inc.](https://www.acorn.io)

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.