package keep

import "testing"

// A cache holds what it was given, replaces a key rather than doubling it,
// and never holds more than it was told to.
func TestItKeepsWhatItCanAndNoMore(t *testing.T) {
	c := New[int, string](3)
	for i := range 3 {
		c.Put(i, "a")
	}
	if n := c.Len(); n != 3 {
		t.Fatalf("holding %d of 3", n)
	}
	c.Put(1, "b")
	if v, ok := c.Get(1); !ok || v != "b" {
		t.Fatalf("key 1 is %q, %v; want b, true", v, ok)
	}
	if n := c.Len(); n != 3 {
		t.Fatalf("replacing a key made it %d", n)
	}
	c.Put(3, "c")
	if n := c.Len(); n != 3 {
		t.Fatalf("one too many made it %d", n)
	}
	if _, ok := c.Get(0); ok {
		t.Error("the oldest is still there")
	}
	for _, k := range []int{1, 2, 3} {
		if _, ok := c.Get(k); !ok {
			t.Errorf("key %d went instead", k)
		}
	}
	c.Clear()
	if n := c.Len(); n != 0 {
		t.Fatalf("cleared, and holding %d", n)
	}
}

// Two keys asked for in turn both stay, which is the whole reason this
// exists: one slot and two cameras threw each other out every frame.
func TestTwoKeysInTurnBothStay(t *testing.T) {
	c := New[string, int](4)
	c.Put("a", 1)
	c.Put("b", 2)
	for range 10 {
		if v, ok := c.Get("a"); !ok || v != 1 {
			t.Fatal("a went")
		}
		if v, ok := c.Get("b"); !ok || v != 2 {
			t.Fatal("b went")
		}
	}
}

// What goes is the least recently used, not the oldest put: a value asked for
// in every round survives any number of new ones.
//
// This is the fixed camera beside a steered one. Every angle the steered
// camera passes through is a new key, and on oldest-put the fixed camera's
// result was thrown out every fourth round though it was wanted in all of
// them.
func TestTheLeastRecentlyUsedGoes(t *testing.T) {
	c := New[int, string](4)
	c.Put(-1, "fixed")
	for i := range 40 {
		if _, ok := c.Get(-1); !ok {
			t.Fatalf("the fixed one went at round %d", i)
		}
		c.Put(i, "moving")
	}
	if _, ok := c.Get(-1); !ok {
		t.Fatal("the fixed one went")
	}
	if n := c.Len(); n != 4 {
		t.Fatalf("holding %d, the most is 4", n)
	}
}

// A cache of nought holds one rather than nothing, so a caller that asks for
// nonsense still gets a working cache.
func TestNoughtHoldsOne(t *testing.T) {
	c := New[int, int](0)
	c.Put(1, 1)
	if _, ok := c.Get(1); !ok {
		t.Fatal("it held nothing at all")
	}
}
