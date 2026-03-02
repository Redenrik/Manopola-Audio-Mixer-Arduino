#include <Wire.h>
#include <util/atomic.h>

#include "i2c_packet.h"

static const uint8_t I2C_ADDR = 0x12;

// Encoder polling state for one rotary encoder + push button.
struct Enc {
  uint8_t pinA, pinB, pinSW;
  uint8_t lastAB;               // 2-bit packed AB state
  volatile int16_t acc;         // quadrature accumulator shared with I2C ISR
  uint8_t btnStable;            // 1 = released (pullup), 0 = pressed
  uint8_t btnLastRead;
  uint32_t btnLastChangeMs;
  volatile uint8_t pressEdge;   // 1 when a press edge is detected
};

static Enc e4{2, 3, 4, 0, 0, 1, 1, 0, 0};
static Enc e5{5, 6, 7, 0, 0, 1, 1, 0, 0};

// Quadrature transition table.
// index = (lastAB << 2) | newAB; value in {-1, 0, +1}
static const int8_t QDEC[16] = {
  0, -1, +1,  0,
  +1, 0,  0, -1,
  -1, 0,  0, +1,
  0, +1, -1, 0
};

static inline uint8_t readAB(const Enc& e) {
  uint8_t a = (uint8_t)digitalRead(e.pinA);
  uint8_t b = (uint8_t)digitalRead(e.pinB);
  return (a << 1) | b;
}

static void initEnc(Enc &e) {
  pinMode(e.pinA, INPUT_PULLUP);
  pinMode(e.pinB, INPUT_PULLUP);
  pinMode(e.pinSW, INPUT_PULLUP);

  e.lastAB = readAB(e);
  e.acc = 0;

  uint8_t sw = (uint8_t)digitalRead(e.pinSW);
  e.btnStable = sw;
  e.btnLastRead = sw;
  e.btnLastChangeMs = millis();
  e.pressEdge = 0;
}

static void updateEnc(Enc &e) {
  uint8_t newAB = readAB(e);
  uint8_t idx = (e.lastAB << 2) | newAB;
  int8_t d = QDEC[idx];
  e.lastAB = newAB;

  if (d != 0) {
    // Keep 16-bit acc updates atomic vs. onI2CRequest ISR reads/writes.
    ATOMIC_BLOCK(ATOMIC_RESTORESTATE) {
      e.acc += d;
    }
  }

  // Simple 15 ms debounce on button.
  uint8_t sw = (uint8_t)digitalRead(e.pinSW);
  if (sw != e.btnLastRead) {
    e.btnLastRead = sw;
    e.btnLastChangeMs = millis();
  } else {
    if ((millis() - e.btnLastChangeMs) > 15 && sw != e.btnStable) {
      e.btnStable = sw;
      if (e.btnStable == 0) {
        e.pressEdge = 1;
      }
    }
  }
}

// Fixed packet on each I2C request:
// byte0 = E4 delta (-127..127) in detent steps
// byte1 = E5 delta
// byte2 = E4 press edge (0/1)
// byte3 = E5 press edge (0/1)
static void onI2CRequest() {
  I2CPacketState s4{e4.acc, e4.pressEdge};
  I2CPacketState s5{e5.acc, e5.pressEdge};
  const I2CPacket packet = i2cBuildPacketAndConsume(s4, s5);

  e4.acc = s4.acc;
  e5.acc = s5.acc;
  e4.pressEdge = s4.pressEdge;
  e5.pressEdge = s5.pressEdge;

  Wire.write((uint8_t)packet.d4);
  Wire.write((uint8_t)packet.d5);
  Wire.write(packet.p4);
  Wire.write(packet.p5);
}

void setup() {
  initEnc(e4);
  initEnc(e5);

  Wire.begin(I2C_ADDR);
  Wire.onRequest(onI2CRequest);
}

void loop() {
  updateEnc(e4);
  updateEnc(e5);
}
