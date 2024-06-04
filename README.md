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

The client allows the caller to run gptscript files, tools, and other operations (see below). There are currently no options for this client, so calling `NewClient()` is all you need. Although, the intention is that a single client is all you need for the life of your application, you should call `Close()` on the client when you are done.

## Options

These are optional options that can be passed to the various `exec` functions.
None of the options is required, and the defaults will reduce the number of calls made to the Model API.

- `cache`: Enable or disable caching. Default (true).
- `cacheDir`: Specify the cache directory.
- `subTool`: Use tool of this name, not the first tool
- `input`: Input arguments for the tool run
- `workspace`: Directory to use for the workspace, if specified it will not be deleted on exit
- `inlcudeEvents`: Whether to include the streaming of events. Default (false). Note that if this is true, you must stream the events. See below for details.
- `chatState`: The chat state to continue, or null to start a new chat and return the state
- `confirm`: Prompt before running potentially dangerous commands
- `prompt`: Allow prompting of the user
- `env`: Extra environment variables to pass to the script in the form `KEY=VAL`

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
	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()
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
	client, err := gptscript.NewClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
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
	client, err := gptscript.NewClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

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
	client, err := gptscript.NewClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

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
	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

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

	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	run, err := client.Evaluate(ctx, gptscript.Options{}, t)
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
	opts := gptscript.Options{
		DisableCache: &[]bool{true}[0],
		Input: "--input hello",
	}

	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

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

	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	run, err := client.Run(ctx, "./hello.gpt", opts)
	if err != nil {
		return err
	}

	for event := range run.Events() {
		// Process event...
	}

	_, err = run.Text()
	return err
}
```

### Confirm

Using the `confirm: true` option allows a user to inspect potentially dangerous commands before they are run. The caller has the ability to allow or disallow their running. In order to do this, a caller should look for the `CallConfirm` event. This also means that `IncludeEvent` should be `true`.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func runFileWithConfirm(ctx context.Context) (string, error) {
	opts := gptscript.Options{
		DisableCache: &[]bool{true}[0],
		Input: "--input hello",
		Confirm: true,
		IncludeEvents: true,
	}

	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	run, err := client.Run(ctx, "./hello.gpt",  opts)
	if err != nil {
		return "", err
	}

	for event := range run.Events() {
		if event.Call != nil && event.Call.Type == gptscript.EventTypeCallConfirm {
			// event.Tool has the information on the command being run.
			// and event.Input will have the input to the command being run.

			err = client.Confirm(ctx, gptscript.AuthResponse{
				ID: event.ID,
				Accept: true, // Or false if not allowed.
				Message: "", // A message explaining why the command is not allowed (ignored if allowed).
			})
			if err != nil {
				// Handle error
			}
		}

		// Process event...
	}

	return run.Text()
}
```

### Prompt

Using the `Prompt: true` option allows a script to prompt a user for input. In order to do this, a caller should look for the `Prompt` event. This also means that `IncludeEvent` should be `true`. Note that if a `Prompt` event occurs when it has not explicitly been allowed, then the run will error.

```go
package main

import (
	"context"

	"github.com/gptscript-ai/go-gptscript"
)

func runFileWithPrompt(ctx context.Context) (string, error) {
	opts := gptscript.Options{
		DisableCache: &[]bool{true}[0],
		Input: "--input hello",
		Prompt: true,
		IncludeEvents: true,
	}

	client, err := gptscript.NewClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	run, err := client.Run(ctx, "./hello.gpt",  opts)
	if err != nil {
		return "", err
	}

	for event := range run.Events() {
		if event.Prompt != nil {
			// event.Prompt has the information to prompt the user.

			err = client.PromptResponse(ctx, gptscript.PromptResponse{
				ID: event.Prompt.ID,
				// Responses is a map[string]string of Fields to values
				Responses: map[string]string{
					event.Prompt.Fields[0]: "Some Value",
				},
			})
			if err != nil {
				// Handle error
			}
		}

		// Process event...
	}

	return run.Text()
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