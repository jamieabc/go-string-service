This program implements a simple http server about string service (uppercase and count).

It listens local port `8080`, with two endpoint:

1. `localhost:8080/uppercase`

    e.g. `curl -XPOST -d'{"s":"hello, world"}' localhost:8080/uppercase`

2. `localhost:8080/count`

    e.g. `curl -XPOST -d'{"s":"hello, world"}' localhost:8080/count`
