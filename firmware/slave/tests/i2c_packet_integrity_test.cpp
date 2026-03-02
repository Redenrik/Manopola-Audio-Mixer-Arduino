#include <cassert>
#include <cstddef>
#include <cstdint>
#include <iostream>

#include "../i2c_packet.h"

static void testPacketLayoutIsExactlyFourBytes() {
  I2CPacket packet{};
  static_assert(sizeof(packet) == 4, "I2C packet must remain 4 bytes");
  assert(sizeof(packet) == 4);
}

static void testClampingAndCarryOverUnderBurstLoad() {
  I2CPacketState e4{4096, 1};
  I2CPacketState e5{-4096, 1};

  const I2CPacket p1 = i2cBuildPacketAndConsume(e4, e5);
  assert(p1.d4 == 127);
  assert(p1.d5 == -127);
  assert(p1.p4 == 1);
  assert(p1.p5 == 1);
  assert(e4.acc == 3588);
  assert(e5.acc == -3588);

  const I2CPacket p2 = i2cBuildPacketAndConsume(e4, e5);
  assert(p2.d4 == 127);
  assert(p2.d5 == -127);
  assert(p2.p4 == 0);
  assert(p2.p5 == 0);
  assert(e4.acc == 3080);
  assert(e5.acc == -3080);
}

static void testDrainMaintainsStepAccounting() {
  I2CPacketState e4{0, 0};
  I2CPacketState e5{0, 0};

  // Simulate heavy runtime load before each I2C request.
  for (int i = 0; i < 2000; ++i) {
    e4.acc += 5;
    e5.acc -= 3;
    if ((i % 31) == 0) {
      e4.pressEdge = 1;
    }

    const I2CPacket packet = i2cBuildPacketAndConsume(e4, e5);
    assert(packet.d4 >= -127 && packet.d4 <= 127);
    assert(packet.d5 >= -127 && packet.d5 <= 127);
    assert(packet.p5 == 0);
  }

  // Residual must stay within less than one detent in either direction.
  assert(e4.acc >= -3 && e4.acc <= 3);
  assert(e5.acc >= -3 && e5.acc <= 3);
}

int main() {
  testPacketLayoutIsExactlyFourBytes();
  testClampingAndCarryOverUnderBurstLoad();
  testDrainMaintainsStepAccounting();
  std::cout << "i2c_packet_integrity_test: PASS\n";
  return 0;
}
