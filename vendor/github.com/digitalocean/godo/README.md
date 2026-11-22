# Godo

[![GitHub Actions CI](https://github.com/digitalocean/godo/actions/workflows/ci.yml/badge.svg)](https://github.com/digitalocean/godo/actions/workflows/ci.yml)
[![GoDoc](https://godoc.org/github.com/digitalocean/godo?status.svg)](https://godoc.org/github.com/digitalocean/godo)

Godo is a Go client library for accessing the DigitalOcean V2 API.

You can view the client API docs here: [http://godoc.org/github.com/digitalocean/godo](http://godoc.org/github.com/digitalocean/godo)

You can view DigitalOcean API docs here: [https://docs.digitalocean.com/reference/api/api-reference/](https://docs.digitalocean.com/reference/api/api-reference/)

> **🚀 New in v1.191.0 — AI & Inference support**
>
> `godo` now ships first-class support for DigitalOcean's
> [DigitalOcean Inference](https://docs.digitalocean.com/products/inference/): chat
> completions (with streaming), image generation, embeddings, batch inference,
> model listing, and more — all from the same `Client`. Jump to
> [**AI & Inference**](#ai--inference) to get started.

## Install
```sh
go get github.com/digitalocean/godo@vX.Y.Z
```

where X.Y.Z is the [version](https://github.com/digitalocean/godo/releases) you need.

or
```sh
go get github.com/digitalocean/godo
```
for non Go modules usage or latest version.

## Usage

```go
import "github.com/digitalocean/godo"
```

Create a new DigitalOcean client, then use the exposed services to
access different parts of the DigitalOcean API.

### Authentication

You can manage API tokens at the DigitalOcean Control Panel
[Applications Page](https://cloud.digitalocean.com/settings/applications).

```go
package main

import (
    "github.com/digitalocean/godo"
)

func main() {
    client := godo.NewFromToken("my-digitalocean-api-token")
}
```

> **Credentials for inference APIs**
>
> What matters is the **credential** you pass to `godo.NewFromToken`, not which
> DigitalOcean API you call:
>
> | What you're calling | What you need |
> | --- | --- |
> | Infrastructure APIs (`Droplets`, `Kubernetes`, `Volumes`, …) | A DigitalOcean API token (PAT). |
> | Inference APIs (`Chat`, `Models`, `Embeddings`, `ImageGenerations`, `Messages`, `Responses`, `BatchInference`, …) | A PAT created with **full access** scope, **or** a **model access key**. |
>
> If you only have a limited-scope PAT, infrastructure calls will work but
> inference calls will fail with `401`. Create a new PAT with full access, or use
> a Model Access Key instead.
>
> ```go
> // Either credential works with godo.NewFromToken:
> client := godo.NewFromToken(os.Getenv("DIGITALOCEAN_TOKEN"))  // full-access PAT
> client := godo.NewFromToken(os.Getenv("MODEL_ACCESS_KEY"))    // model access key
> ```

If you need to provide a `context.Context` to your new client, you should use [`godo.NewClient`](https://godoc.org/github.com/digitalocean/godo#NewClient) to manually construct a client instead.

## Action Gateway

`client.ActionGateway` exposes the Action Gateway control-plane resources: tools,
toolbelts, output views, MCP servers, connections, provider credentials,
sessions, endpoints, users, and end-user limits. Use a PAT with the corresponding `action_gateway` scopes. For example:

```go
toolbelts, _, err := client.ActionGateway.Toolbelts.List(ctx, &godo.ActionGatewayToolbeltsListOptions{Status: "active"})
if err != nil { return err }
fmt.Println(toolbelts.Toolbelts)

session, _, err := client.ActionGateway.Sessions.CreateRuntime(ctx, &godo.ActionGatewaySessionCreateRequest{
    ActorID: "alice",
})
if err != nil { return err }
tools, _, err := session.ChatTools(ctx, nil) // action_search, action_invoke, action_code
if err != nil { return err }
completion, _, err := client.Chat.Completions.New(ctx, &godo.ChatCompletionNewParams{
    Model: "llama3.3-70b-instruct", Messages: []godo.ChatCompletionMessage{godo.UserMessage("Find recent news")}, Tools: tools,
})
if err != nil { return err }
toolMessages, _, err := session.HandleChatToolCalls(ctx, completion)
if err != nil { return err }
fmt.Println(toolMessages)
```

`session.Tools` supports direct discovery and invocation, while `session.Code`
runs sandboxed code. `MessageTools`/`HandleMessageToolCalls` and
`ResponseTools`/`HandleResponseToolCalls` integrate with the other inference
formats. `CreateRuntime` defaults to an ask-before-executing policy; provide an
explicit policy to change it. Omitted `Tools` enables all tools, while an empty
non-nil slice enables none. MCP calls accept only HTTPS session URLs on
`actions.do-ai.run`; `WithActionGatewayMCPBaseURL` overrides the origin for
development or tests.

## AI & Inference

> Talk to models on [DigitalOcean Inference](https://docs.digitalocean.com/products/inference/) with the same `godo.Client`.

The [Serverless Inference API](https://docs.digitalocean.com/reference/api/reference/serverless-inference/) is available at `https://inference.do-ai.run/`. Use a **DigitalOcean PAT with full access scope** or a **model access key** — see the [credentials note](#authentication) above.

#### Chat completion

```go
completion, _, err := client.Chat.Completions.New(ctx, &godo.ChatCompletionNewParams{
    Model: "llama3.3-70b-instruct",
    Messages: []godo.ChatCompletionMessage{
        godo.UserMessage("Write me a haiku"),
    },
})
```

#### List models

```go
page, _, err := client.Models.List(ctx)
for _, m := range page.Data {
    fmt.Println(m.ID)
}
```

#### Image generation

```go
image, _, err := client.ImageGenerations.Generate(ctx, &godo.ImageGenerateParams{
    Model:  "stable-diffusion-3.5-large",
    Prompt: "A friendly cartoon shark typing on a laptop at a sunny beach",
    N:      1,
})
```

For streaming, embeddings, messages, responses, async invocations, batch inference, agent inference, and full runnable programs, see [`examples/serverless-inference/`](examples/serverless-inference/) and [`examples/agent-inference/`](examples/agent-inference/).

## Sandboxes

The `sandbox` package creates and manages DigitalOcean Sandboxes (`/v2/sandboxes`). It reads the token from `DIGITALOCEAN_ACCESS_TOKEN`, or you can set `Config.Token`.

```go
import (
    "context"

    "github.com/digitalocean/godo/sandbox"
)

ctx := context.Background()

c, err := sandbox.NewClient(nil)
if err != nil {
    log.Fatal(err)
}

sb, err := c.Create(ctx, nil) // template coding-base, size mars-2vcpu-4gb
if err != nil {
    log.Fatal(err)
}
defer sb.Kill(ctx)

err = sb.Pause(ctx)
err = sb.Resume(ctx)
```

Every field of `CreateParams` is optional:

```go
off := false
sb, err := c.Create(ctx, &sandbox.CreateParams{
    Template:            "coding-base",
    Size:                "mars-2vcpu-4gb",
    AllowInternetAccess: &off,                       // false limits egress to the platform file-transfer host
    EnvVars:             map[string]string{"MY_KEY": "value"}, // values are secrets
})
```

Find existing sandboxes:

```go
sb, err := c.Get(ctx, id)      // fetch by ID
sb, err := c.Connect(ctx, id)  // fetch, and resume it if it is paused

for sb, err := range c.List(&sandbox.ListParams{State: godo.SandboxStateRunning}).All(ctx) {
    if err != nil { ... }
    fmt.Println(sb.ID(), sb.Template())
}

pages := c.List(&sandbox.ListParams{Limit: 50}) // or one page at a time
for pages.HasNext() {
    items, err := pages.NextItems(ctx)
    ...
}
```

`sb.Refresh(ctx)` re-reads a sandbox, `sb.IsRunning(ctx)` checks its state, and `sb.WaitUntilRunning(ctx, nil)` polls until it is running. `c.Kill(ctx, id)` and `c.Pause(ctx, id)` work by ID; `Kill` reports `false` when the sandbox is already gone.

Failed calls return a `*godo.ErrorResponse`. Test it with `sandbox.IsNotFound`, `IsUnauthorized`, `IsPermissionDenied`, `IsInvalidRequest`, `IsConflict` and `IsRateLimited`.

For the raw API, use `client.Sandboxes` (`List`, `Get`, `Create`, `Delete`, `Pause`, `Resume`) on a regular `godo.Client`. A runnable program is in [`examples/sandbox/`](examples/sandbox/).

Run commands with `sb.Commands`:

```go
// A shell command (bash -lc): wait for it and read the result.
res, err := sb.Commands.Run(ctx, `echo "Hello"`, &sandbox.RunOptions{
    Cwd:      "/workspace",
    OnStdout: func(text string) { fmt.Print(text) }, // optional, as output arrives
})
fmt.Println(res.Stdout, res.ExitCode) // a non-zero exit code is returned, not an error

// An executable and its arguments: start it, then attach, write to it or stop it.
cmd, err := sb.Commands.Start(ctx, "cat", &sandbox.StartOptions{Stdin: true})
err = cmd.WriteStdin(ctx, []byte("hello\n"), true) // true closes stdin
for ev, err := range cmd.Events(ctx) {            // started, stdout, stderr, exit
    if err != nil { ... }
    if ev.Type == sandbox.EventStdout { fmt.Print(ev.Text) }
}

running, err := sb.Commands.List(ctx)                // PID, Cmd, Cwd
err = sb.Commands.Connect(running[0].PID).Kill(ctx) // SIGTERM; Signal(ctx, godo.SandboxCommandSignalKill) to force
```

Build and manage custom templates:

```go
// Build a template from an OCI image (rebased onto a base template, coding-base by default).
tpl, err := c.CreateTemplate(ctx, &sandbox.TemplateParams{
    Name:         "my-template",
    SourceOCIRef: "registry.digitalocean.com/my-team/my-image:v1",
})
err = tpl.WaitUntilReady(ctx, nil)           // the build is asynchronous; fails if it fails
sb, err := c.Create(ctx, &sandbox.CreateParams{Template: tpl.Name()})

for t, err := range c.ListTemplates(nil).All(ctx) { ... }
err = tpl.Update(ctx, &sandbox.TemplateUpdate{SourceOCIRef: "registry.digitalocean.com/my-team/my-image:v2"})
builds, err := tpl.Builds(0).NextItems(ctx)  // []godo.SandboxTemplateBuild, with Status and Error
deleted, err := tpl.Delete(ctx)              // false if it is already gone
```

The raw calls are `client.Sandboxes.CreateTemplate`, `GetTemplate`, `ListTemplates`, `UpdateTemplate`, `DeleteTemplate`, `ListTemplateBuilds` and `GetTemplateBuild`.

Take and manage snapshots of a running or paused sandbox:

```go
// Name and TTL are optional. The capture is asynchronous.
snap, err := sb.CreateSnapshot(ctx, &sandbox.SnapshotParams{Name: "my-snapshot", TTL: "168h"})
err = snap.WaitUntilReady(ctx, nil) // fails, with the API's message, if the snapshot failed

for s, err := range c.ListSnapshots(&sandbox.ListSnapshotsParams{SandboxID: sb.ID()}).All(ctx) { ... }
again, err := c.GetSnapshot(ctx, snap.ID())   // again.Info.Size() is the size in bytes once complete
deleted, err := snap.Delete(ctx)               // false if it is already gone
```

The raw calls are `client.Sandboxes.CreateSnapshot`, `GetSnapshot`, `ListSnapshots` and `DeleteSnapshot`. A snapshot captures the boot disk and memory, not the workspace data disk. The API does not yet have a way to start a sandbox from a snapshot.

The raw calls are `client.Sandboxes.StartCommand`, `ListCommands`, `StreamCommand`, `SignalCommand` and `WriteCommandStdin`. Output can only be read while a command runs: a command that exits before the stream attaches fails with a 404 (`sandbox.IsNotFound`). File transfer is not available yet: it needs routes that are not in the API.

## Examples


To create a new Droplet:

```go
dropletName := "super-cool-droplet"

createRequest := &godo.DropletCreateRequest{
    Name:   dropletName,
    Region: "nyc3",
    Size:   "s-1vcpu-1gb",
    Image: godo.DropletCreateImage{
        Slug: "ubuntu-20-04-x64",
    },
}

ctx := context.TODO()

newDroplet, _, err := client.Droplets.Create(ctx, createRequest)

if err != nil {
    fmt.Printf("Something bad happened: %s\n\n", err)
    return err
}
```

### Pagination

If a list of items is paginated by the API, you must request pages individually. For example, to fetch all Droplets:

```go
func DropletList(ctx context.Context, client *godo.Client) ([]godo.Droplet, error) {
    // create a list to hold our droplets
    list := []godo.Droplet{}

    // create options. initially, these will be blank
    opt := &godo.ListOptions{}
    for {
        droplets, resp, err := client.Droplets.List(ctx, opt)
        if err != nil {
            return nil, err
        }

        // append the current page's droplets to our list
        list = append(list, droplets...)

        // if we are at the last page, break out the for loop
        if resp.Links == nil || resp.Links.IsLastPage() {
            break
        }

        page, err := resp.Links.CurrentPage()
        if err != nil {
            return nil, err
        }

        // set the page we want for the next request
        opt.Page = page + 1
    }

    return list, nil
}
```

Some endpoints offer token based pagination. For example, to fetch all Registry Repositories:

```go
func ListRepositoriesV2(ctx context.Context, client *godo.Client, registryName string) ([]*godo.RepositoryV2, error) {
    // create a list to hold our registries
    list := []*godo.RepositoryV2{}

    // create options. initially, these will be blank
    opt := &godo.TokenListOptions{}
    for {
        repositories, resp, err := client.Registry.ListRepositoriesV2(ctx, registryName, opt)
        if err != nil {
            return nil, err
        }

        // append the current page's registries to our list
        list = append(list, repositories...)

        // if we are at the last page, break out the for loop
        if resp.Links == nil || resp.Links.IsLastPage() {
            break
        }

        // grab the next page token
        nextPageToken, err := resp.Links.NextPageToken()
        if err != nil {
            return nil, err
        }

        // provide the next page token for the next request
        opt.Token = nextPageToken
    }

    return list, nil
}
```

### Automatic Retries and Exponential Backoff

The Godo client can be configured to use automatic retries and exponentional backoff for requests that fail with 429 or 500-level response codes via [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp). To configure Godo to enable usage of go-retryablehttp, the `RetryConfig.RetryMax` must be set.

```go
tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{
    AccessToken: "dop_v1_xxxxxx",
})

oauth_client := oauth2.NewClient(oauth2.NoContext, tokenSrc)

waitMax := godo.PtrTo(6.0)
waitMin := godo.PtrTo(3.0)

retryConfig := godo.RetryConfig{
    RetryMax:     3,
    RetryWaitMin: waitMin,
    RetryWaitMax: waitMax,
}

client, err := godo.New(oauth_client, godo.WithRetryAndBackoffs(retryConfig))
```

Please refer to the [RetryConfig Godo documentation](https://pkg.go.dev/github.com/digitalocean/godo#RetryConfig) for more information.

## Versioning

Each version of the client is tagged and the version is updated accordingly.

To see the list of past versions, run `git tag`.


## Documentation

For a comprehensive list of examples, check out the [API documentation](https://docs.digitalocean.com/reference/api/api-reference/#tag/SSH-Keys).

For details on all the functionality in this library, see the [GoDoc](http://godoc.org/github.com/digitalocean/godo) documentation.


## Contributing

We love pull requests! Please see the [contribution guidelines](CONTRIBUTING.md).
