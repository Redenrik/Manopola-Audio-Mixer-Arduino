#ifndef MAMA_FIRMWARE_SLAVE_I2C_PACKET_H
#define MAMA_FIRMWARE_SLAVE_I2C_PACKET_H

#include <stdint.h>

struct I2CPacketState {
  int16_t acc;
  uint8_t pressEdge;
};

struct I2CPacket {
  int8_t d4;
  int8_t d5;
  uint8_t p4;
  uint8_t p5;
};

static inline int8_t i2cPacketStepsFromAccumulator(int16_t acc) {
  int16_t steps = acc / 4; // 4 quadrature transitions = 1 detent step
  if (steps > 127) steps = 127;
  if (steps < -127) steps = -127;
  return (int8_t)steps;
}

static inline I2CPacket i2cBuildPacketAndConsume(I2CPacketState &e4, I2CPacketState &e5) {
  const int8_t d4 = i2cPacketStepsFromAccumulator(e4.acc);
  const int8_t d5 = i2cPacketStepsFromAccumulator(e5.acc);

  e4.acc -= (int16_t)d4 * 4;
  e5.acc -= (int16_t)d5 * 4;

  const uint8_t p4 = e4.pressEdge ? 1 : 0;
  const uint8_t p5 = e5.pressEdge ? 1 : 0;
  e4.pressEdge = 0;
  e5.pressEdge = 0;

  return I2CPacket{d4, d5, p4, p5};
}

#endif
