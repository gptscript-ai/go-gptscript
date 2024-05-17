# go-gptscript

This module provides a set of functions to interact with gptscripts. It allows for executing scripts, listing available tools and models, and more.

## Installation

To use this module, you need to have Go installed on your system. Then, you can install the module via:

```bash
go get github.com/gptscript-ai/go-gptscript
```

## Usage

To use the module, you need to first set the OPENAI_API_KEY environment variable to your OpenAI API key.

Additionally, you need the `gptscript` binary. You can install it on your system using the [installation instructions](https://github.com/gptscript-ai/gptscript?tab=readme-ov-file#1-install-the-latest-release). The binary can be on the PATH, or the `GPTSCRIPT_BIN` environment variable can be used to specify its location.

## Client

There are currently a couple "global" options, and the client helps to manage those. A client without any options is
likely what you want. However, here are the current global options:

- `gptscriptURL`: The URL (including `http(s)://) of an "SDK server" to use instead of the fork/exec model.
- `gptscriptBin`: The path to a `gptscript` binary to use instead of the bundled one.

## Options

These are optional options that can be passed to the various `exec` functions.
None of the options is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching. Default (true).
- `cacheDir`: Specify the cache directory.
- `quiet`: No output logging
- `chdir`: Change current working directory
- `subTool`: Use tool of this name, not the first tool
- `input`: Input arguments for the tool run
- `workspace`: Directory to use for the workspace, if specified it will not be deleted on exit
- `inlcudeEvents`: Whether to include the streaming of events. Default (false). Note that if this is true, you must stream the events. See below for details.
- `chatState`: The chat state to continue, or null to start a new chat and return the state

## Functions

### listTools

Lists all the available built-in tools.

**Usage:**

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func listTools(ctx context.Context) (string, error) {
	client := gptscript.NewClient(gptscript.ClientOpts{})
	return client.ListTools(ctx)
}
```

### listModels

Lists all the available models, returns a list.

**Usage:**

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func listModels(ctx context.Context) ([]string, error) {
	client := gptscript.NewClient(gptscript.ClientOpts{})
	return client.ListModels(ctx)
}
```

### Parse

Parse file into a Tool data structure

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func parse(ctx context.Context, fileName string) ([]gptscript.Node, error) {
	client := gptscript.NewClient(gptscript.ClientOpts{})
	return client.Parse(ctx, fileName)
}
```

### ParseTool

Parse contents that represents a GPTScript file into a data structure.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func parseTool(ctx context.Context, contents string) ([]gptscript.Node, error) {
	client := gptscript.NewClient(gptscript.ClientOpts{})
	return client.ParseTool(ctx, contents)
}
```

### Fmt

Parse convert a tool data structure into a GPTScript file.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func parse(ctx context.Context, nodes []gptscript.Node) (string, error) {
	client := gptscript.NewClient(gptscript.ClientOpts{})
	return client.Fmt(ctx, nodes)
}
```

### Evaluate

Executes a tool with optional arguments.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func runTool(ctx context.Context) (string, error) {
	t := gptscript.ToolDef{
		Instructions: "who was the president of the united states in 1928?",
	}

	client := gptscript.NewClient(gptscript.ClientOpts{})
	run, err := client.Evaluate(ctx, gptscript.Opts{}, t)
	if err != nil {
		return "", err
	}

	return run.Text()
}
```

### Run

Executes a GPT script file with optional input and arguments. The script is relative to the callers source directory.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func runFile(ctx context.Context) (string, error) {
	opts := gptscript.Opts{
		DisableCache: &[]bool{true}[0],
		Input: "--input hello",
	}

	client := gptscript.NewClient(gptscript.ClientOpts{})
	run, err := client.Run(ctx, "./hello.gpt",  opts)
	if err != nil {
		return "", err
	}

	return run.Text()
}
```

### Streaming events

In order to stream events, you must set `IncludeEvents` option to `true`. You if you don't set this and try to stream events, then it will succeed, but you will not get any events. More importantly, if you set `IncludeEvents` to `true`, you must stream the events for the script to complete.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func streamExecTool(ctx context.Context) error {
	opts := gptscript.Opts{
		DisableCache:  &[]bool{true}[0],
		IncludeEvents: true,
		Input:         "--input world",
	}

	client := gptscript.NewClient(gptscript.ClientOpts{})
	run, err := client.Run(ctx, "./hello.gpt", opts)
	if err != nil {
		return err
	}

	for event := range run.Events() {
		// Process event...
	}

	// Wait for the output to ensure the script completes successfully.
	_, err = run.Text()
	return err
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

## License

Copyright (c) 2024, [Acorn Labs, Inc.](https://www.acorn.io)

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

<http://www.apache.org/licenses/LICENSE-2.0>

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.