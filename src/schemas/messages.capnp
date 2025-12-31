@0xcf4762d38e91a0b1;

using Go = import "go.capnp";
$Go.package("schemas");
$Go.import("github.com/Bastien-Antigravity/safe-socket/src/schemas");

# Standard Handshake Message (TCP / Connection Establishment)
# Connection works also through proxies, thanks to HelloMsg
struct HelloMsg {
  fromName     @0 :Text;
  fromHost     @1 :Text;
  fromAddress  @2 :Text;
  toAddress    @3 :Text;
  fromPublicIP @4 :Text;
}

# Optimized Stateless Envelope (UDP Per-Packet)
struct PacketEnvelope {
  senderID @0 :Text;
  payload  @1 :Data;
}
