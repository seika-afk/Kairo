package ot

import "testing"

func TestApplyInsert(t *testing.T) {
	doc := []rune("abc")
	op := Op{
		Type:     "insert",
		Position: 3,
		Text:     "X",
	}

	got := Apply(doc, op)
	want := []rune("abcX")

	if string(got) != string(want) {
		t.Fatalf("Apply insert = %q, want %q", string(got), string(want))
	}
}

func TestApplyDelete(t *testing.T) {
	doc := []rune("hello")
	op := Op{
		Type:     "delete",
		Position: 1,
		Length:   2,
	}

	got := Apply(doc, op)
	want := []rune("hlo")

	if string(got) != string(want) {
		t.Fatalf("Apply delete = %q, want %q", string(got), string(want))
	}
}

func TestApplyClampsOutOfRangeDelete(t *testing.T) {
	doc := []rune("a")
	op := Op{
		Type:     "delete",
		Position: 2,
		Length:   1,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Apply panicked on out-of-range delete: %v", r)
		}
	}()

	got := Apply(doc, op)
	want := []rune("a")

	if string(got) != string(want) {
		t.Fatalf("Apply out-of-range delete = %q, want %q", string(got), string(want))
	}
}

func TestTransformDeleteAgainstInsertUsesTextLength(t *testing.T) {
	incoming := Op{
		Type:     "delete",
		Position: 5,
		Length:   2,
	}
	against := Op{
		Type:     "insert",
		Position: 2,
		Text:     "abcd",
	}

	got := Transform(incoming, against)

	if got.Position != 1 {
		t.Fatalf("Transform delete/insert position = %d, want %d", got.Position, 1)
	}
}

func TestTransformInsertAgainstInsert(t *testing.T) {
	incoming := Op{
		Type:     "insert",
		Position: 3,
		Text:     "x",
	}
	against := Op{
		Type:     "insert",
		Position: 1,
		Text:     "ab",
	}

	got := Transform(incoming, against)

	if got.Position != 5 {
		t.Fatalf("Transform insert/insert position = %d, want %d", got.Position, 5)
	}
}

func TestTransformInsertAgainstDelete(t *testing.T) {
	incoming := Op{
		Type:     "insert",
		Position: 5,
		Text:     "x",
	}
	against := Op{
		Type:     "delete",
		Position: 2,
		Length:   2,
	}

	got := Transform(incoming, against)

	if got.Position != 3 {
		t.Fatalf("Transform insert/delete position = %d, want %d", got.Position, 3)
	}
}

func TestTransformDeleteAgainstInsert(t *testing.T) {
	incoming := Op{
		Type:     "delete",
		Position: 5,
		Length:   2,
	}
	against := Op{
		Type:     "insert",
		Position: 2,
		Length:   2,
		Text:     "ab",
	}

	got := Transform(incoming, against)

	if got.Position != 3 {
		t.Fatalf("Transform delete/insert position = %d, want %d", got.Position, 3)
	}
}

func TestTransformDeleteAgainstDelete(t *testing.T) {
	incoming := Op{
		Type:     "delete",
		Position: 6,
		Length:   2,
	}
	against := Op{
		Type:     "delete",
		Position: 2,
		Length:   2,
	}

	got := Transform(incoming, against)

	if got.Position != 4 {
		t.Fatalf("Transform delete/delete position = %d, want %d", got.Position, 4)
	}
}

func TestTransformAgainstHistory(t *testing.T) {
	history := []Op{
		{
			Type:     "insert",
			Position: 1,
			Text:     "ab",
			Length:   2,
		},
		{
			Type:     "delete",
			Position: 4,
			Length:   1,
		},
	}

	incoming := Op{
		Type:     "insert",
		Position: 2,
		Text:     "x",
	}

	got := TransformAgainstHistory(incoming, history, 0)

	if got.Position != 4 {
		t.Fatalf("TransformAgainstHistory position = %d, want %d", got.Position, 4)
	}
}

func TestTransformAgainstHistorySinceIndex(t *testing.T) {
	history := []Op{
		{
			Type:     "insert",
			Position: 1,
			Text:     "ab",
			Length:   2,
		},
		{
			Type:     "delete",
			Position: 4,
			Length:   1,
		},
	}

	incoming := Op{
		Type:     "insert",
		Position: 2,
		Text:     "x",
	}

	got := TransformAgainstHistory(incoming, history, 1)

	if got.Position != 2 {
		t.Fatalf("TransformAgainstHistory since=1 position = %d, want %d", got.Position, 2)
	}
}
