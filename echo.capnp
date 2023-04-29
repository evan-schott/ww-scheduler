using Go = import "/go.capnp";

@0x8d728f3bbdc2c9fe;

$Go.package("lb");
$Go.import("lb");

# Declare the Echo capability, which provides single method.
interface Echo {
    echo @0 (payload :Data) -> (response :Data);
}