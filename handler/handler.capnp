using Go = import "/go.capnp";

@0xb8ace6235fa2dd9f;

$Go.package("handler");
$Go.import("handler");

# Declare the Handler capability, which provides single method.
interface Handler {
    handle @0 (request :Text) -> (response :Text);
}