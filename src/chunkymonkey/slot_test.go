package slot

import (
	"fmt"
	"testing"

	"chunkymonkey/itemtype"
	. "chunkymonkey/types"
)

func slotEq(s1, s2 *Slot) bool {
	return (s1.ItemType == s2.ItemType &&
		s1.Count == s2.Count &&
		s1.Data == s2.Data)
}

type slotTest struct {
	desc                 string
	initialA, initialB   Slot
	expectedA, expectedB Slot
	expectedChange       bool
}

func (test *slotTest) run(t *testing.T, op func(a, b *Slot) bool) {
	t.Logf("Test %s:", test.desc)
	t.Logf("    initial   a=%+v, b=%+v", test.initialA, test.initialB)
	t.Logf("    expecting a=%+v, b=%+v, changed=%t",
		test.expectedA, test.expectedB,
		test.expectedChange)

	// Sanity check the test itself. Sum of inputs must equal sum of
	// outputs.
	sumInput := test.initialA.Count + test.initialB.Count
	sumExpectedOutput := test.expectedA.Count + test.expectedB.Count
	if sumInput != sumExpectedOutput {
		t.Errorf(
			"    Test incorrect: sum of inputs %d != sum of expected outputs %d",
			sumInput, sumExpectedOutput,
		)
		return
	}

	a := test.initialA
	b := test.initialB
	changed := op(&a, &b)
	if !slotEq(&test.expectedA, &a) || !slotEq(&test.expectedB, &b) {
		t.Errorf("    Fail: got a=%+v, b=%+v", a, b)
	}
	if test.expectedChange != changed {
		t.Errorf("    Fail: got changed=%t", changed)
	}
}

func runTests(t *testing.T, tests []slotTest, op func(a, b *Slot) bool) {
	for i := range tests {
		tests[i].run(t, op)
	}
}

func makeItemType(id ItemTypeId) *itemtype.ItemType {
	return &itemtype.ItemType{
		Id:       id,
		Name:     fmt.Sprintf("<Test ItemType #%d>", id),
		MaxStack: 64,
		ToolType: 0,
		ToolUses: 0,
	}
}

// Tests cases that are common to both Slot.Add and Slot.AddWhole.
func TestSlot_Add_Common(t *testing.T) {
	apple := makeItemType(1)
	orange := makeItemType(2)

	tests := []slotTest{
		{
			"one empty slot added to another",
			Slot{nil, 0, 0}, Slot{nil, 0, 0},
			Slot{nil, 0, 0}, Slot{nil, 0, 0},
			false,
		},
		// Tests involving the same item types: (or empty plus an item)
		{
			"1 + 0 => 1 + 0",
			Slot{apple, 1, 0}, Slot{nil, 0, 0},
			Slot{apple, 1, 0}, Slot{nil, 0, 0},
			false,
		},
		{
			"1 + 1 => 2 + 0",
			Slot{apple, 1, 0}, Slot{apple, 1, 0},
			Slot{apple, 2, 0}, Slot{nil, 0, 0},
			true,
		},
		{
			"0 + 20 => 20 + 0",
			Slot{nil, 0, 0}, Slot{apple, 20, 0},
			Slot{apple, 20, 0}, Slot{nil, 0, 0},
			true,
		},
		{
			"0 + 64 => 64 + 0",
			Slot{nil, 0, 0}, Slot{apple, 64, 0},
			Slot{apple, 64, 0}, Slot{nil, 0, 0},
			true,
		},
		{
			"65 + 1 => 65 + 1 (already above max count)",
			Slot{apple, 65, 0}, Slot{apple, 1, 0},
			Slot{apple, 65, 0}, Slot{apple, 1, 0},
			false,
		},
		{
			"64 + 64 => 64 + 64",
			Slot{apple, 64, 0}, Slot{apple, 64, 0},
			Slot{apple, 64, 0}, Slot{apple, 64, 0},
			false,
		},
		{
			"1 + 1 => 1 + 1 where items' \"Data\" value differs",
			Slot{apple, 1, 5}, Slot{apple, 1, 6},
			Slot{apple, 1, 5}, Slot{apple, 1, 6},
			false,
		},
		{
			"1 + 1 => 2 + 0 where items' \"Data\" value is the same",
			Slot{apple, 1, 5}, Slot{apple, 1, 5},
			Slot{apple, 2, 5}, Slot{nil, 0, 0},
			true,
		},
		{
			"0 + 1 => 1 + 0 - carrying the \"Data\" value",
			Slot{nil, 0, 0}, Slot{apple, 1, 5},
			Slot{apple, 1, 5}, Slot{nil, 0, 0},
			true,
		},
		// Tests involving different item types:
		{
			"different item types don't mingle",
			Slot{apple, 5, 0}, Slot{orange, 5, 0},
			Slot{apple, 5, 0}, Slot{orange, 5, 0},
			false,
		},
	}

	t.Log("Testing Add")
	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.Add(b)
		},
	)

	t.Log("Testing AddWhole")
	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.AddWhole(b)
		},
	)
}

func TestSlot_Add(t *testing.T) {
	apple := makeItemType(1)

	tests := []slotTest{
		{
			"32 + 33 => 64 + 1 (hitting max count)",
			Slot{apple, 32, 0}, Slot{apple, 33, 0},
			Slot{apple, 64, 0}, Slot{apple, 1, 0},
			true,
		},
	}

	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.Add(b)
		},
	)
}

func TestSlot_AddWhole(t *testing.T) {
	apple := makeItemType(1)

	tests := []slotTest{
		{
			"32 + 33 => 32 + 33 (hitting max count)",
			Slot{apple, 32, 0}, Slot{apple, 33, 0},
			Slot{apple, 32, 0}, Slot{apple, 33, 0},
			false,
		},
	}

	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.AddWhole(b)
		},
	)
}

func TestSlot_Swap(t *testing.T) {
	apple := makeItemType(1)
	orange := makeItemType(2)

	tests := []slotTest{
		{
			"swapping unequal slots",
			Slot{apple, 2, 3}, Slot{orange, 5, 6},
			Slot{orange, 5, 6}, Slot{apple, 2, 3},
			true,
		},
		{
			"swapping equal slots",
			Slot{apple, 2, 3}, Slot{apple, 2, 3},
			Slot{apple, 2, 3}, Slot{apple, 2, 3},
			false,
		},
	}

	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.Swap(b)
		},
	)
}

func TestSlot_Split(t *testing.T) {
	apple := makeItemType(1)
	orange := makeItemType(2)

	tests := []slotTest{
		// No-op tests.
		{
			"splitting an empty slot",
			Slot{nil, 0, 0}, Slot{nil, 0, 0},
			Slot{nil, 0, 0}, Slot{nil, 0, 0},
			false,
		},
		{
			"splitting from a non-empty slot to another non-empty",
			Slot{apple, 2, 0}, Slot{apple, 3, 0},
			Slot{apple, 2, 0}, Slot{apple, 3, 0},
			false,
		},
		{
			"splitting from an empty slot to a non-empty",
			Slot{nil, 0, 0}, Slot{apple, 3, 0},
			Slot{nil, 0, 0}, Slot{apple, 3, 0},
			false,
		},
		{
			"splitting from a non-empty slot to another non-empty with" +
				" incompatible types",
			Slot{apple, 2, 0}, Slot{orange, 3, 0},
			Slot{apple, 2, 0}, Slot{orange, 3, 0},
			false,
		},
		// Remaining tests should all result in changes. They all take from a
		// non-empty subject slot and into the src empty slot.
		{
			"splitting even-numbered stack",
			Slot{apple, 64, 0}, Slot{nil, 0, 0},
			Slot{apple, 32, 0}, Slot{apple, 32, 0},
			true,
		},
		{
			"splitting odd-numbered stack",
			Slot{apple, 5, 0}, Slot{nil, 0, 0},
			Slot{apple, 2, 0}, Slot{apple, 3, 0},
			true,
		},
		{
			"splitting single-item stack",
			Slot{apple, 1, 0}, Slot{nil, 0, 0},
			Slot{nil, 0, 0}, Slot{apple, 1, 0},
			true,
		},
		{
			"item type and data copy",
			Slot{apple, 64, 5}, Slot{nil, 0, 0},
			Slot{apple, 32, 5}, Slot{apple, 32, 5},
			true,
		},
		{
			"item type and data move",
			Slot{apple, 1, 5}, Slot{nil, 0, 0},
			Slot{nil, 0, 0}, Slot{apple, 1, 5},
			true,
		},
	}

	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.Split(b)
		},
	)
}

func TestSlot_AddOne(t *testing.T) {
	apple := makeItemType(1)
	orange := makeItemType(2)

	tests := []slotTest{
		// No-op tests.
		{
			"adding from empty to empty",
			Slot{nil, 0, 0}, Slot{nil, 0, 0},
			Slot{nil, 0, 0}, Slot{nil, 0, 0},
			false,
		},
		{
			"adding from empty to non-empty",
			Slot{apple, 1, 0}, Slot{nil, 0, 0},
			Slot{apple, 1, 0}, Slot{nil, 0, 0},
			false,
		},
		{
			"adding incompatible types",
			Slot{apple, 1, 0}, Slot{orange, 4, 0},
			Slot{apple, 1, 0}, Slot{orange, 4, 0},
			false,
		},
		{
			"adding incompatible data values",
			Slot{apple, 1, 0}, Slot{apple, 1, 2},
			Slot{apple, 1, 0}, Slot{apple, 1, 2},
			false,
		},
		{
			"adding to an already full stack",
			Slot{apple, 64, 0}, Slot{apple, 10, 0},
			Slot{apple, 64, 0}, Slot{apple, 10, 0},
			false,
		},
		// Remaining tests should all result in changes. They all take from a
		// non-empty src slot into a compatible subject slot.
		{
			"adding item to empty, copies type and data",
			Slot{nil, 0, 0}, Slot{apple, 3, 0},
			Slot{apple, 1, 0}, Slot{apple, 2, 0},
			true,
		},
		{
			"adding item to non-empty",
			Slot{apple, 5, 0}, Slot{apple, 3, 0},
			Slot{apple, 6, 0}, Slot{apple, 2, 0},
			true,
		},
		{
			"adding item to non-empty, empties src",
			Slot{apple, 5, 2}, Slot{apple, 1, 2},
			Slot{apple, 6, 2}, Slot{nil, 0, 0},
			true,
		},
	}

	runTests(
		t, tests,
		func(a, b *Slot) bool {
			return a.AddOne(b)
		},
	)
}
