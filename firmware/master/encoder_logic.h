#pragma once

#include <stdint.h>

struct EncoderState {
  uint8_t lastAB;
  int16_t acc;
  uint8_t btnStable;
  uint8_t btnLastRead;
  uint32_t btnLastChangeMs;
  uint8_t pressEdge;
};

static const int8_t ENCODER_QDEC[16] = {
  0, -1, +1,  0,
  +1, 0,  0, -1,
  -1, 0,  0, +1,
  0, +1, -1, 0,
};

static inline void encoderInitState(EncoderState &state, uint8_t initialAB, uint8_t initialSW, uint32_t nowMs) {
  state.lastAB = initialAB;
  state.acc = 0;
  state.btnStable = initialSW;
  state.btnLastRead = initialSW;
  state.btnLastChangeMs = nowMs;
  state.pressEdge = 0;
}

static inline void encoderUpdateState(EncoderState &state, uint8_t newAB, uint8_t sw, uint32_t nowMs, uint16_t debounceMs = 15) {
  uint8_t idx = (state.lastAB << 2) | newAB;
  int8_t d = ENCODER_QDEC[idx];
  state.lastAB = newAB;
  if (d != 0) {
    state.acc += d;
  }

  if (sw != state.btnLastRead) {
    state.btnLastRead = sw;
    state.btnLastChangeMs = nowMs;
  } else if ((uint32_t)(nowMs - state.btnLastChangeMs) > debounceMs && sw != state.btnStable) {
    state.btnStable = sw;
    if (state.btnStable == 0) {
      state.pressEdge = 1;
    }
  }
}

static inline int8_t encoderConsumeSteps(EncoderState &state) {
  int8_t steps = (int8_t)(state.acc / 4);
  state.acc -= (int16_t)steps * 4;
  return steps;
}
