Hangar provides a local daemon for running Fly.io servers locally in Go.

## Usage

To start the package at `./demo`, run:

```bash
$ go run ./bin -p ./demo
```

This starts a daemon with small number of machines all running in different 'regions'.
It performs basic load-balancing between them (with "the user" assumed to be in the 1st region), or respects [the `fly-prefer-region` header](https://fly.io/docs/reference/dynamic-request-routing/).

You can demonstrate having multiple jobs run with:

```bash
$ curl http://localhost:8080/info -H "fly-prefer-region: ams"
$ curl http://localhost:8080/info -H "fly-prefer-region: syd"
```

The code inside `./lib` helps provide a layer that hides local development vs. the real Fly deployed environment.

You can use it in your code like:

```go
import (
  hangar "github.com/samthor/hangar/lib"
)

func main() {
  // find out about ourselves
  self := hangar.Self()

  // do something with other instances
  others, err := hangar.Discover(context.Background())

  // serve on the local $PORT
  log.Fatal(http.ListenAndServe(hangar.ListenPort(), nil))
}

```

### Start Multiple Instances

You can use `curl` to kick off a few instances:

```bash
$
```

## Extensions/TODOs

This just discovers instances in the same process group (in production).
It doesn't even know/care about process groups locally.

This currently runs Go packages, but really, it could run any command N times&mdash;Hangar just sets `$PORT` and other environment variables.

Assumes that the package under control stops after some time (does not kill it when idle).
If a process exits, it's not restarted for any reason (this is different than Fly production, which restarts all non-zero codes).
