@0xcf4762d38e91a0b1;

using Go = import "go.capnp";
$Go.package("schemas");
$Go.import("github.com/Bastien-Antigravity/safe-socket/src/schemas");

# Standard Handshake Message (TCP / Connection Establishment)
struct HelloMsg {
  name     @0 :Text;
  host     @1 :Text;
  address  @2 :Text;
  publicIP @3 :Text;
}

# Optimized Stateless Envelope (UDP Per-Packet)
struct PacketEnvelope {
  senderID @0 :Text;
  payload  @1 :Data;
}
