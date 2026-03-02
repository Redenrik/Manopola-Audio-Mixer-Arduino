#include <cassert>
#include <cstdint>
#include <iostream>

#include "../encoder_logic.h"

static void applyForwardDetent(EncoderState &state, uint32_t &nowMs, uint8_t sw = 1) {
  const uint8_t seq[] = {0b00, 0b10, 0b11, 0b01, 0b00};
  for (size_t i = 1; i < sizeof(seq); ++i) {
    nowMs += 1;
    encoderUpdateState(state, seq[i], sw, nowMs);
  }
}

static void testFastSpinAccumulatesBurst() {
  EncoderState state{};
  uint32_t nowMs = 0;
  encoderInitState(state, 0b00, 1, nowMs);

  for (int i = 0; i < 24; ++i) {
    applyForwardDetent(state, nowMs);
  }

  const int8_t steps = encoderConsumeSteps(state);
  assert(steps == 24);
  assert(encoderConsumeSteps(state) == 0);
}

static void testButtonBounceYieldsSinglePressEdge() {
  EncoderState state{};
  uint32_t nowMs = 100;
  encoderInitState(state, 0b00, 1, nowMs);

  // noisy edge within debounce window
  nowMs += 1;
  encoderUpdateState(state, 0b00, 0, nowMs);
  nowMs += 1;
  encoderUpdateState(state, 0b00, 1, nowMs);
  nowMs += 1;
  encoderUpdateState(state, 0b00, 0, nowMs);
  assert(state.pressEdge == 0);

  // stable low past debounce threshold emits one edge
  nowMs += 20;
  encoderUpdateState(state, 0b00, 0, nowMs);
  assert(state.pressEdge == 1);

  // additional stable low updates should not emit repeated edges
  state.pressEdge = 0;
  nowMs += 20;
  encoderUpdateState(state, 0b00, 0, nowMs);
  assert(state.pressEdge == 0);

  // release and press again should generate a new edge
  nowMs += 20;
  encoderUpdateState(state, 0b00, 1, nowMs);
  nowMs += 20;
  encoderUpdateState(state, 0b00, 1, nowMs);
  nowMs += 1;
  encoderUpdateState(state, 0b00, 0, nowMs);
  nowMs += 20;
  encoderUpdateState(state, 0b00, 0, nowMs);
  assert(state.pressEdge == 1);
}

int main() {
  testFastSpinAccumulatesBurst();
  testButtonBounceYieldsSinglePressEdge();
  std::cout << "encoder_logic_stress_test: PASS\n";
  return 0;
}
