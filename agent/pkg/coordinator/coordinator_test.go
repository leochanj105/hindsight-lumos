package coordinator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func (c *Coordinator) known_at(id TriggerID, addr string) bool {
	trigger, trigger_exists := c.triggers[id]
	if trigger_exists {
		_, known_at_addr := trigger.known_at[addr]
		return known_at_addr
	}
	return false
}

func (c *Coordinator) trace_known_at(id uint64, addr string) bool {
	trace, trace_exists := c.traces[id]
	if trace_exists {
		_, known_at_addr := trace.known_at[addr]
		return known_at_addr
	}
	return false
}

func (c *Coordinator) addr_count(id TriggerID) int {
	trigger, trigger_exists := c.triggers[id]
	if trigger_exists {
		return len(trigger.known_at)
	}
	return 0
}

func (c *Coordinator) trace_addr_count(id uint64) int {
	trace, trace_exists := c.traces[id]
	if trace_exists {
		return len(trace.known_at)
	}
	return 0
}

func TestCoordinator(t *testing.T) {
	assert := assert.New(t)

	var c Coordinator
	c.Init()

	triggerid := TriggerID{1, uint64(75)}

	addrs := c.AddTrigger("a", Trigger{triggerid, []uint64{uint64(75)}})
	assert.Equal(0, len(addrs), "First trigger returns no addrs")
	assert.Equal(1, len(c.triggers), "Trigger was created")
	assert.Equal(1, c.addr_count(triggerid), "Trigger is known at 1 address")
	assert.True(c.known_at(triggerid, "a"), "Trigger is known at a")

	addrs = c.AddTrigger("a", Trigger{triggerid, []uint64{uint64(76)}})
	assert.Equal(0, len(addrs), "Updating a trigger from the same source doesn't change anything")
	assert.Equal(1, c.addr_count(triggerid), "Trigger is known at 1 address")
	assert.True(c.known_at(triggerid, "a"), "Trigger is known at a")

	addrs = c.AddTrigger("b", Trigger{triggerid, []uint64{uint64(77)}})
	assert.True(c.known_at(triggerid, "a"), "Trigger is known at a")
	assert.True(c.known_at(triggerid, "b"), "Trigger is known at b")
	assert.Equal(2, c.addr_count(triggerid), "Trigger is known at 2 addresses")
	assert.Equal([]string{"a"}, addrs, "Updating a trigger from a different source requires redistribution to first source")

	disseminate := c.AddBreadcrumb("a", Breadcrumbs{uint64(75), []string{"c"}})
	assert.Equal(1, len(disseminate), "Adding a breadcrumb c->d requires dissemination to c")

	disseminate = c.AddBreadcrumb("c", Breadcrumbs{uint64(75), []string{"d"}})
	assert.Equal(1, len(disseminate), "Adding a breadcrumb c->d requires dissemination to d")

	disseminate = c.AddBreadcrumb("c", Breadcrumbs{uint64(77), []string{"d"}})
	assert.Equal(0, len(disseminate), "Adding another breadcrumb c->d does not require dissemination to d")
}

func TestCoordinatorAfterBreadcrumbs(t *testing.T) {
	assert := assert.New(t)

	var c Coordinator
	c.Init()

	triggerid := TriggerID{1, uint64(75)}

	disseminate := c.AddBreadcrumb("a", Breadcrumbs{uint64(75), []string{"b"}})
	assert.Equal(0, len(disseminate), "Adding breadcrumbs requires no dissemination yet")
	assert.True(c.trace_known_at(uint64(75), "a"), "Trace is known at a")
	assert.True(c.trace_known_at(uint64(75), "b"), "Trace is known at b")

	disseminate = c.AddBreadcrumb("b", Breadcrumbs{uint64(75), []string{"c"}})
	assert.Equal(0, len(disseminate), "Adding breadcrumbs requires no dissemination yet")
	assert.True(c.trace_known_at(uint64(75), "a"), "Trace is known at a")
	assert.True(c.trace_known_at(uint64(75), "b"), "Trace is known at b")
	assert.True(c.trace_known_at(uint64(75), "c"), "Trace is known at c")

	disseminate = c.AddBreadcrumb("c", Breadcrumbs{uint64(75), []string{"d"}})
	assert.Equal(0, len(disseminate), "Adding breadcrumbs requires no dissemination yet")
	assert.True(c.trace_known_at(uint64(75), "a"), "Trace is known at a")
	assert.True(c.trace_known_at(uint64(75), "b"), "Trace is known at b")
	assert.True(c.trace_known_at(uint64(75), "c"), "Trace is known at c")
	assert.True(c.trace_known_at(uint64(75), "d"), "Trace is known at d")
	assert.False(c.trace_known_at(uint64(77), "a"), "Trace 77 not known yet")

	disseminate = c.AddBreadcrumb("a", Breadcrumbs{uint64(77), []string{"e"}})
	assert.Equal(0, len(disseminate), "Adding breadcrumbs requires no dissemination yet")
	assert.True(c.trace_known_at(uint64(77), "a"), "Trace 77 is known at a")
	assert.True(c.trace_known_at(uint64(77), "e"), "Trace 77 is known at e")

	disseminate = c.AddBreadcrumb("e", Breadcrumbs{uint64(77), []string{"f"}})
	assert.Equal(0, len(disseminate), "Adding breadcrumbs requires no dissemination yet")
	assert.True(c.trace_known_at(uint64(77), "a"), "Trace 77 is known at a")
	assert.True(c.trace_known_at(uint64(77), "e"), "Trace 77 is known at e")
	assert.True(c.trace_known_at(uint64(77), "f"), "Trace 77 is known at f")

	addrs := c.AddTrigger("a", Trigger{triggerid, []uint64{uint64(75)}})
	assert.Equal(3, len(addrs), "Trigger must be disseminated to 3 other breadcrumbs")
	assert.Equal(1, len(c.triggers), "Trigger was created")
	assert.Equal(4, c.addr_count(triggerid), "Trigger is known at 4 addresses")
	assert.True(c.known_at(triggerid, "a"), "Trigger is known at a")
	assert.True(c.known_at(triggerid, "b"), "Trigger is known at b")
	assert.True(c.known_at(triggerid, "c"), "Trigger is known at c")
	assert.True(c.known_at(triggerid, "d"), "Trigger is known at d")

	addrs = c.AddTrigger("b", Trigger{triggerid, []uint64{uint64(77)}})
	assert.Equal(5, len(addrs), "Trigger must be redisseminated to 5 known breadcrumbs")
	assert.Equal(1, len(c.triggers), "Trigger was created")
	assert.Equal(6, c.addr_count(triggerid), "Trigger is known at 6 addresses")
	assert.Equal(c.trace_addr_count(uint64(75)), 4, "Trace 75 known at 4 addresses")
	assert.Equal(c.trace_addr_count(uint64(77)), 4, "Trace 77 known at 3 addresses")

}
