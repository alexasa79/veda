package terminal

import (
	"reflect"
	"testing"
)

func TestDetectScrollOperation(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		previous := []uint64{1, 2, 3, 4, 5, 99}
		current := []uint64{2, 3, 4, 5, 6, 99}

		op, ok := detectScrollOperation(current, previous, 4)
		if !ok {
			t.Fatal("expected upward scroll")
		}
		want := scrollOperation{top: 0, bottom: 4, lines: 1}
		if op != want {
			t.Fatalf("got %+v, want %+v", op, want)
		}
	})

	t.Run("down", func(t *testing.T) {
		previous := []uint64{1, 2, 3, 4, 5, 99}
		current := []uint64{0, 1, 2, 3, 4, 99}

		op, ok := detectScrollOperation(current, previous, 4)
		if !ok {
			t.Fatal("expected downward scroll")
		}
		want := scrollOperation{top: 0, bottom: 4, lines: -1}
		if op != want {
			t.Fatalf("got %+v, want %+v", op, want)
		}
	})

	t.Run("unchanged repeated rows", func(t *testing.T) {
		previous := []uint64{1, 1, 1, 1, 1, 99}
		if _, ok := detectScrollOperation(previous, previous, 4); ok {
			t.Fatal("unchanged rows must not be detected as scrolling")
		}
	})

	t.Run("short match", func(t *testing.T) {
		previous := []uint64{1, 2, 3, 4, 5, 99}
		current := []uint64{2, 3, 30, 40, 50, 99}
		if _, ok := detectScrollOperation(current, previous, 4); ok {
			t.Fatal("short shifted run must not be detected as scrolling")
		}
	})
}

func TestApplyScrollToFrontBuffer(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		front := []string{"a", "b", "c", "d", "status"}
		applyScrollToFrontBuffer(front, scrollOperation{top: 0, bottom: 3, lines: 1})
		want := []string{"b", "c", "d", "\x00", "status"}
		if !reflect.DeepEqual(front, want) {
			t.Fatalf("got %q, want %q", front, want)
		}
	})

	t.Run("down", func(t *testing.T) {
		front := []string{"a", "b", "c", "d", "status"}
		applyScrollToFrontBuffer(front, scrollOperation{top: 0, bottom: 3, lines: -1})
		want := []string{"\x00", "a", "b", "c", "status"}
		if !reflect.DeepEqual(front, want) {
			t.Fatalf("got %q, want %q", front, want)
		}
	})
}
