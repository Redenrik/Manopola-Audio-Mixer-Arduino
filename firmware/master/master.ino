#include <Wire.h>

#include "encoder_logic.h"

static const uint8_t I2C_ADDR = 0x12;

// -------- Encoder polling (no librerie) --------
struct Enc {
  uint8_t pinA, pinB, pinSW;
  EncoderState state;
};

static Enc e1{2, 3, 4, {0, 0, 1, 1, 0, 0}};
static Enc e2{5, 6, 7, {0, 0, 1, 1, 0, 0}};
static Enc e3{8, 9, 10, {0, 0, 1, 1, 0, 0}};


static inline uint8_t readAB(const Enc& e) {
  uint8_t a = (uint8_t)digitalRead(e.pinA);
  uint8_t b = (uint8_t)digitalRead(e.pinB);
  return (a << 1) | b;
}

static void initEnc(Enc &e) {
  pinMode(e.pinA, INPUT_PULLUP);
  pinMode(e.pinB, INPUT_PULLUP);
  pinMode(e.pinSW, INPUT_PULLUP);

  uint8_t sw = (uint8_t)digitalRead(e.pinSW);
  encoderInitState(e.state, readAB(e), sw, millis());
}

static void updateEnc(Enc &e) {
  uint8_t newAB = readAB(e);
  uint8_t sw = (uint8_t)digitalRead(e.pinSW);
  encoderUpdateState(e.state, newAB, sw, millis());
}

static int8_t consumeSteps(Enc &e) {
  // 4 transizioni = 1 step
  return encoderConsumeSteps(e.state);
}

static void emitEncDelta(uint8_t encId, int8_t delta) {
  if (delta == 0) return;
  // manda uno o più step (se giri veloce)
  // esempio: E2:+3
  Serial.print("E");
  Serial.print(encId);
  Serial.print(":");
  if (delta > 0) Serial.print("+");
  Serial.println((int)delta);
}

static void emitButtonPress(uint8_t encId) {
  Serial.print("B");
  Serial.print(encId);
  Serial.println(":1"); // pressione = toggle mute lato PC
}

void setup() {
  Serial.begin(115200);
  Wire.begin(); // Master I2C

  // Unique project handshake used by host auto-detection
  Serial.println("MAMA:HELLO:1");

  initEnc(e1);
  initEnc(e2);
  initEnc(e3);
}

void loop() {
  // --- aggiorna encoder locali ---
  updateEnc(e1);
  updateEnc(e2);
  updateEnc(e3);

  int8_t d1 = consumeSteps(e1);
  int8_t d2 = consumeSteps(e2);
  int8_t d3 = consumeSteps(e3);

  emitEncDelta(1, d1);
  emitEncDelta(2, d2);
  emitEncDelta(3, d3);

  if (e1.state.pressEdge) { e1.state.pressEdge = 0; emitButtonPress(1); }
  if (e2.state.pressEdge) { e2.state.pressEdge = 0; emitButtonPress(2); }
  if (e3.state.pressEdge) { e3.state.pressEdge = 0; emitButtonPress(3); }

  // --- poll I2C slave (E4, E5 + press) ---
  Wire.requestFrom((int)I2C_ADDR, 4);
  if (Wire.available() >= 4) {
    int8_t d4 = (int8_t)Wire.read();
    int8_t d5 = (int8_t)Wire.read();
    uint8_t p4 = Wire.read();
    uint8_t p5 = Wire.read();

    emitEncDelta(4, d4);
    emitEncDelta(5, d5);
    if (p4) emitButtonPress(4);
    if (p5) emitButtonPress(5);
  }

  // piccola pausa per non saturare la seriale (puoi ridurre/levare se vuoi)
  delay(2);
}
