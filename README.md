# easymcp

`easymcp` is a tiny helper for spinning up Model Context Protocol (MCP) tool servers using nothing more than a YAML file. It maps simple command line programs to MCP tools so you can experiment with custom tooling quickly.

**This project is experimental and has not been production tested.** It is meant for local prototyping and demos. It does not implement a full MCP runtime or any of the robustness features you would expect from a mature server implementation.

## What it provides

- Read a YAML definition of tools
- Launch an MCP server over stdio so the tools can be called by any MCP client
- Support basic input schema generation for tool arguments

## What it does not provide

- Persistent storage, memory or session management
- Authentication, authorization or encryption
- Process supervision or resource isolation
- Any guarantee of stability or security

If you need a production-ready solution you should look at a hardened MCP server or use `mcpo` (see below) to proxy to standard HTTP tooling.

## Installation

You need a working Go installation. Then run:

```bash
go install github.com/kevin-cantwell/easymcp/cmd/easymcp@latest
```

This installs the `easymcp` binary into your `$GOBIN` directory.

## Usage

Create a YAML file describing your tools, for example:

```yaml
tools:
  # Namespaces result in tool names like "utils/echo"
  - namespace: utils
    name: echo
    description: Echo a message
    run:
      # The command to run, which must be available in the server's environment
      cmd: echo
      args:
        - "{{.message}}"
    input:
      - name: message
        # Type values are a subset of JSON Schema types: string, number, integer, or boolean
        type: string
        description: message to echo
        required: true
        # If provided, the input will be limited to these values
        enum:
          - "foo"
          - "bar"
    output:
      # The output format can be audio, image, or text. Text is used if unspecified.
      format: text
```

Start the MCP server using that file:

```bash
easymcp --config tools.yaml
```

The server communicates over stdio and can be embedded or proxied by another tool. One option is [`mcpo`](https://github.com/open-webui/mcpo), which exposes an MCP server as an OpenAPI HTTP service. Running `mcpo` alongside `easymcp` lets you generate ready‑to‑use tools for chat LLM products such as [Open WebUI](https://github.com/open-webui/open-webui) in just a few commands.

## License

This project is provided under the MIT license.
