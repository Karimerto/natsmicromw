Here are two simple examples of different types of middleware functions.

Simply build the examples with `go build` and run. Send a request with the NATS client (e.g. `nats req svc.echo "Hello"`) and watch the magic happen.

 * `duration` is an example of measuring a request duration.
 * `microreq` is an example of data modification using the `MicroRequest` and `MicroReply`.
