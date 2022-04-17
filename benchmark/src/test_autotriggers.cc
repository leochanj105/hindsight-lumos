#include <iostream>
#include <cassert>
#include <vector>

#include "hindsight/autotriggers.h"

extern "C" {
  #include "common.h"
}

void test_categorytrigger() {
  CategoryTrigger t(0.1);

  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("hello") == false);
  assert(t.addSample("world") == true);
  assert(t.addSample("world") == false);
  assert(t.addSample("world") == false);
  assert(t.addSample("world") == false);
}

void test_filtertrigger_inclusive() {
  std::set<std::string> filters = {"hello", "world"};
  FilterTrigger t(filters, true);

  assert(t.shouldTrigger("hello") == true);
  assert(t.shouldTrigger("world") == true);
  assert(t.shouldTrigger("goodbye") == false);
}

void test_filtertrigger_exclusive() {
  std::set<std::string> filters = {"hello", "world"};
  FilterTrigger t(filters, false);

  assert(t.shouldTrigger("hello") == false);
  assert(t.shouldTrigger("world") == false);
  assert(t.shouldTrigger("goodbye") == true);
}

void test_percentiletrigger() {
  PercentileTrigger<uint64_t> t(0.9);

  for (int i = 0; i < 10; i++) {
    assert(t.addSample(i) == false);
  }
  assert(t.addSample(5) == false);
  assert(t.addSample(4) == false);
  assert(t.addSample(20) == true);
  assert(t.addSample(5) == false);
  assert(t.addSample(4) == false);
  assert(t.addSample(20) == true);
  assert(t.addSample(5) == false);
  assert(t.addSample(4) == false);
  assert(t.addSample(19) == true);
  assert(t.addSample(5) == false);
  assert(t.addSample(4) == false);
  assert(t.addSample(18) == false);
}

void test_percentiletrigger_perf() {
  std::vector<double> percentiles = {0.9, 0.99, 0.999};

  for (auto percentile : percentiles) {
    PercentileTrigger<uint64_t> t(percentile);

    uint64_t begin = nanos();

    int iterations = 1000000;
    for (int i = 0; i < iterations; i++) {
      t.addSample(i);
    }
    uint64_t end = nanos();

    double duration = (end - begin) / (double) iterations;

    std::cout << "percentiletrigger " << percentile << ": " << duration << "ns per addSample" << std::endl;
  }
}

void test_exception_trigger() {
  bool trigger_fired = false;
  std::function<void(void)> callback = [&trigger_fired] () -> void {
    trigger_fired = true;
  };

  try {
    ExceptionTrigger et(callback);

    std::string("abc").substr(10); // throws std::length_error

  } catch (const std::exception& e) {
    // Ignore exception
  }
  assert(trigger_fired);


  trigger_fired = false;
  try {
    ExceptionTrigger et(callback);

    std::string("abc").substr(2); // nothrow

  } catch (const std::exception& e) {
    // Ignore exception
  }
  assert(!trigger_fired);
}

void test_triggerset() {
  TriggerSet ts(5);

  for (int i = 7; i < 12; i++) {
    ts.addTrace(i);
  }

  auto& cur = ts.get();
  assert(cur.size() == 5);
  for (int i = 7; i < 12; i++) {
    bool found = 0;
    for (auto trace_id : cur) {
      if (trace_id == i) {
        found = true;
      }
    }
    assert(found);
  }

  ts.addTrace(77);
  cur = ts.get();
  for (auto trace_id : cur) {
    assert(trace_id != 7);
  }
  for (int i = 8; i < 12; i++) {
    bool found = false;
    for (auto trace_id : cur) {
      if (trace_id == i) {
        found = true;
      }
    }
    assert(found);
  }

}

int main (int argc, char **argv) {
  test_categorytrigger();
  test_filtertrigger_inclusive();
  test_filtertrigger_exclusive();
  test_percentiletrigger();
  test_percentiletrigger_perf();
  test_exception_trigger();
  test_triggerset();
  std::cout << "All tests passed" << std::endl;
}