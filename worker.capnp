using Go = import "/go.capnp";

@0x95c0f3edf8dd266c;

$Go.package("worker");
$Go.import("worker");

# Declare the Echo capability, which provides single method.
interface Worker {
    assign @0 (wasm :Data, input :Int64, difficulty :Int64) -> (result :Text);
}