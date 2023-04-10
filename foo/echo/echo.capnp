using Go = import "/go.capnp";

@0x8d728f3bbdc2c9fe;

$Go.package("echo");
$Go.import("echo");

# Declare the Echo capability, which provides single method.
interface Echo {
    send @0 (msg :Text) -> (response :Text);
}