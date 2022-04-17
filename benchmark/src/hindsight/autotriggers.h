/*
 * Copyright 2022 Max Planck Institute for Software Systems *
 */

#pragma once
#ifndef SRC_AUTOTRIGGERS_H_
#define SRC_AUTOTRIGGERS_H_

#include <set>
#include <map>
#include <string>/* These two files are included for the Order Statistics Tree. */
#include <ext/pb_ds/assoc_container.hpp>
#include <ext/pb_ds/tree_policy.hpp>
#include <functional>

template <typename T> 
class SlidingWindowT {
private:
	unsigned window_size;

	/* An order statistics tree is used to implement a wrapper around a C++
	   set with the ability to know the ordinal number of an item in the set
	   and also to get an item by its ordinal number from the set.
	   The data structure I use is implemented in STL but only for GNU C++.
	   Some sources are documented below:
	     -- https://gcc.gnu.org/onlinedocs/libstdc++/ext/pb_ds/
	     -- https://codeforces.com/blog/entry/11080
	     -- https://gcc.gnu.org/onlinedocs/libstdc++/ext/pb_ds/tree_based_containers.html
	     -- https://opensource.apple.com/source/llvmgcc42/llvmgcc42-2336.9/libstdc++-v3/testsuite/ext/pb_ds/example/tree_order_statistics.cc.auto.html
		 -- https://stackoverflow.com/questions/44238144/order-statistics-tree-using-gnu-pbds-for-multiset
	     -- https://www.geeksforgeeks.org/order-statistic-tree-using-fenwick-tree-bit/ */

	typedef __gnu_pbds::tree<
		T,
		__gnu_pbds::null_type,
		std::less_equal<T>,
		__gnu_pbds::rb_tree_tag,
		__gnu_pbds::tree_order_statistics_node_update> OrderedMultiset;

	/* We maintain a list of data items (FIFO ordered) so that the latest
	   and the oldest items can be easily tracked for insertion and removal.
	   And we also maintain a parallel OrderedMultiset data structure where the
	   items are stored in an order statistics tree so that querying, say, the
	   99th percentile value is easy. We also maintain an upper bound on sliding
	   window size. After the first few iterations, the number of data items
	   is always equal to the upper bound. Thus, we have:
			-- Invariant 1: q.size() == oms.size()
			-- Invariant 2: q.size() <= window_size */
	std::list<T> q;
	OrderedMultiset oms;

public:
	SlidingWindowT() : window_size(100) {}
	SlidingWindowT(unsigned window_size) : window_size(window_size) {}

	/* Assumption: q.size() == oms.size() */
	unsigned get_size() { return q.size(); }

	/* Requirement: rank < oms.size() */
	T get_value(unsigned rank) { return (*(oms.find_by_order(rank))); }
	T get_percentile(float percentile) {
		float position = percentile * (q.size() - 1);
		unsigned up = ceil(position);
		unsigned down = floor(position);
		if (up == down) return get_value(up);
		return get_value(up) * (position - down) + get_value(down) * (up - position);
	}
	T get_min() { return get_value(0); }
	T get_max() { return get_value(q.size()-1); }
	void insert(T latest) {
		q.push_back(latest);
		oms.insert(latest);
		if (q.size() > window_size) {
			uint64_t oldest = q.front();
			q.pop_front();
			auto it=oms.upper_bound (oldest);
			oms.erase(it); // Assumption: *it == oldest
		}
	}
	
	SlidingWindowT(unsigned window_size, T initial_value) : window_size(window_size) {
		insert(initial_value);
	}
};


/* Trigger on basic filter */
class FilterTrigger {
public:
  bool inclusive;
  std::set<std::string> labels;

  FilterTrigger(std::set<std::string> labels, bool inclusive) : labels(labels), inclusive(inclusive) {
  }

  bool shouldTrigger(std::string label) {
    if (inclusive) {
      return labels.find(label) != labels.end();
    } else {
      return labels.find(label) == labels.end();
    }
  }

};

class CategoryTrigger {
public:
  std::map<std::string, uint64_t> counts;
  uint64_t total;
  double threshold;

  CategoryTrigger(double threshold) : threshold(threshold), total(0) {}

  bool addSample(std::string label) {
    uint64_t frequency = ++this->counts[label];
    uint64_t total = ++this->total;
    uint64_t required_count = threshold * (double) total;
    return frequency <= required_count;
  }

};

template <typename T> 
class PercentileTrigger {
public:
  double percentile;
  uint64_t windowsize;
  SlidingWindowT<T> window;
  uint64_t samples;

  PercentileTrigger(double percentile) : percentile(percentile), windowsize((unsigned) (1.0 / (1.0 - percentile))), window(2*windowsize), samples(0) {
  }

  bool addSample(T value) {
    bool shouldTrigger = false;
    if (samples >= windowsize) {
      T p = window.get_percentile(percentile);
      shouldTrigger = p <= value;
    }
    window.insert(value);
    samples++;
    return shouldTrigger;
  }

};

class ExceptionTrigger {
public:
  std::function<void(void)> callback;
  ExceptionTrigger(std::function<void(void)> callback) : callback(callback) {}
  ~ExceptionTrigger() {
    if(std::uncaught_exception()) {
      callback();
    }
  }
};

class TriggerSet {
public:
  std::vector<uint64_t> trace_ids;
  uint64_t head;
  uint64_t tail;
  TriggerSet(size_t size) : trace_ids(size), head(0), tail(0) {}

  void addTrace(uint64_t trace_id) {
    trace_ids[tail] = trace_id;
    tail = (tail + 1) % trace_ids.size();
    if (tail == head) {
      head = (head + 1) % trace_ids.size();
    }
  }

  std::vector<uint64_t>& get() {
    return trace_ids;
  }
};

#endif  // SRC_AUTOTRIGGERS_H_
