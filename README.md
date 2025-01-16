# Railpack Go

_Huge work in progress_

## Todo

- [x] Setup architecture for creating build plan based on user code
- [x] Convert plan to LLB
- [x] Build LLB with a Buildkit client
- [x] Buildkit frontend that can be used as an image
- [ ] Optimized build plan to LLB generation
- [ ] Solidify build plan configuration
- [ ] Support configuring Railpack with a config file

## Architecture

- Analyze user code to create a build plan
- Convert build plan to LLB
- `build`
  - Send LLB to BuildKit over GRPC
  - Pipe output to `docker load` (hidden)
- Custom frontend
  - Buildkit invokes our frontend image with a mounted context
  - We look for a `rpk.json` file, which is a serialized build plan
  - We convert the build plan to LLB and return it

## Usage

Railpack can currently be used to build an image with BuildKit directly, or as a custom BuildKit frontend.

### Building directly

Railpack will instantiate a BuildKit client and communicate to over GRPC in order to build the generated LLB.

```bash
go run cmd/cli/main.go --verbose build examples/node-bun
```

You need to have a BuildKit instance running (see below).

### Custom frontend

A custom frontend allows us to build the build plan and serialize into a
`rpk.json` file. At a later time, we can use this file to build an image by invoking `buildctl`.

One downside of this approach is that the frontend needs to be hosted in a
public image registry. Currently we are using ghcr.io
[here](https://github.com/railwayapp/railpack-go/pkgs/container/railpack-go).

Build a plan for you app first:

```bash
# Save to the test/ directory, but this can be anywhere
go run cmd/cli/main.go --verbose plan examples/node-bun --out test/rpk.json
```

Now we can build the plan against an app directory:

```bash
buildctl build \
  --local context=examples/node-bun \ # The app directory to build
  --local dockerfile=test \ # The path to the `rpk.json` file
  --frontend=gateway.v0 \ # Specify that we are using a custom frontend
  --opt source=ghcr.io/railwayapp/railpack-go:railpack-frontend \ # The frontend image to use
  --output type=docker,name=test \ # The output type and name
  | docker load # Pipe into docker load to get a docker image we can run
```

To update the frontend image, you can run

```bash
mise run build-and-push-frontend
```

### Mise commands

```bash
# Lint and format
mise run check

# Run tests
mise run test
```

### BuildKit setup

If building with the `build` command, you need to have a BuildKit instance
running with the `BUILDKIT_HOST` environment variable set to the container.

```bash
docker run --rm --privileged -d --name buildkit moby/buildkit

# Set the buildkit host to the container
export BUILDKIT_HOST=docker-container://buildkit
```
