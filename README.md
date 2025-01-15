# Railpack Go

_Huge work in progress_

## Todo

- [x] Setup architecture for creating build plan based on user code
- [x] Convert plan to LLB
- [ ] Build LLB with a Buildkit client
- [ ] Buildkit frontend that can be used as an image
- [ ] Lots of other stuff

## Usage

Only works on node and it only runs install for various package managers.

`build` outputs Buildkit LLB that can be piped into `buildctl` to run a build.
The output can be piped into `docker load` to get a docker image.

```bash
go run cmd/cli/main.go --verbose build examples/node-bun \
  | buildctl build --local context=examples/node-bun --output type=docker,name=node \
  | docker load
```
