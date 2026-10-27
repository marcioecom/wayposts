package post

import "testing"

func TestCoalesceKeepsLastWritePerPair(t *testing.T) {
	events := []likeEvent{
		{userID: 1, postID: 10, liked: true},
		{userID: 1, postID: 10, liked: false},
		{userID: 2, postID: 10, liked: false},
		{userID: 2, postID: 10, liked: true},
		{userID: 3, postID: 10, liked: true},
		{userID: 3, postID: 11, liked: true},
	}

	inserts, deletes := coalesce(events)

	if len(deletes) != 1 {
		t.Fatalf("esperava 1 delete, veio %d: %+v", len(deletes), deletes)
	}
	if deletes[0].userID != 1 || deletes[0].postID != 10 {
		t.Fatalf("delete inesperado: %+v", deletes[0])
	}

	if len(inserts) != 3 {
		t.Fatalf("esperava 3 inserts, veio %d: %+v", len(inserts), inserts)
	}
	want := map[[2]int64]bool{
		{2, 10}: true,
		{3, 10}: true,
		{3, 11}: true,
	}
	for _, e := range inserts {
		if !want[[2]int64{e.userID, e.postID}] {
			t.Fatalf("insert inesperado: %+v", e)
		}
	}
}

func TestCoalesceEmpty(t *testing.T) {
	inserts, deletes := coalesce(nil)
	if len(inserts) != 0 || len(deletes) != 0 {
		t.Fatalf("esperava listas vazias, veio inserts=%v deletes=%v", inserts, deletes)
	}
}
