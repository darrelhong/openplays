package listener

import (
	"context"
	"fmt"
	"testing"
	"time"

	"openplays/server/internal/db"
)

// SpyMessageStore records all calls made to it, allowing tests to verify
// behaviour (what was called, in what order, with what arguments) rather
// than implementation details.
type SpyMessageStore struct {
	Calls    []string                      // operation log: "GetRecent", "Insert"
	messages []db.GetRecentMessageTextsRow // simulate stored messages
	nextID   int64

	// Error injection
	GetRecentErr error
	InsertErr    error
}

func (s *SpyMessageStore) GetRecentMessageTexts(_ context.Context, _ time.Time) ([]db.GetRecentMessageTextsRow, error) {
	s.Calls = append(s.Calls, "GetRecent")
	if s.GetRecentErr != nil {
		return nil, s.GetRecentErr
	}
	return s.messages, nil
}

func (s *SpyMessageStore) InsertRawMessage(_ context.Context, arg db.InsertRawMessageParams) (db.RawMessage, error) {
	s.Calls = append(s.Calls, "Insert")
	if s.InsertErr != nil {
		return db.RawMessage{}, s.InsertErr
	}
	s.nextID++
	// Store so subsequent dedupe checks see this message
	s.messages = append(s.messages, db.GetRecentMessageTextsRow{
		ID:          s.nextID,
		MessageText: arg.MessageText,
	})
	return db.RawMessage{ID: s.nextID}, nil
}

func TestHandleMessage(t *testing.T) {

	t.Run("new message is inserted", func(t *testing.T) {
		store := &SpyMessageStore{}

		result, err := HandleMessage(context.Background(), store, "telegram", "Daniel",
			"Looking for HB players at Bedok, 3 Apr 8pm", time.Now(), nil, nil)

		assertNoError(t, err)
		assertResult(t, result, HandleInserted)
		assertCalls(t, store.Calls, []string{"GetRecent", "Insert"})
	})

	t.Run("exact duplicate is skipped", func(t *testing.T) {
		store := &SpyMessageStore{}
		msg := "Looking for HB players at Bedok, 3 Apr 8pm, $10, RSL Supreme"

		HandleMessage(context.Background(), store, "telegram", "Daniel", msg, time.Now(), nil, nil)

		result, err := HandleMessage(context.Background(), store, "telegram", "Daniel", msg, time.Now(), nil, nil)

		assertNoError(t, err)
		assertResult(t, result, HandleSkipped)
		// First call: GetRecent + Insert. Second call: GetRecent only (no Insert).
		assertCalls(t, store.Calls, []string{"GetRecent", "Insert", "GetRecent"})
	})

	t.Run("near duplicate with slot count change is skipped", func(t *testing.T) {
		store := &SpyMessageStore{}

		original := `Looking for friendly HB-LI players for wkend games
🗓️ 4 Apr, Sat
📍 Kuo Chuan Presbyterian Primary
⌚ 4pm - 6pm
🏸 HB-LI
3 slot left
$10 per pax, RSL Supreme`

		repost := `Looking for friendly HB-LI players for wkend games
🗓️ 4 Apr, Sat
📍 Kuo Chuan Presbyterian Primary
⌚ 4pm - 6pm
🏸 HB-LI
2 slot left
$10 per pax, RSL Supreme`

		HandleMessage(context.Background(), store, "telegram", "Daniel", original, time.Now(), nil, nil)
		result, err := HandleMessage(context.Background(), store, "telegram", "Daniel", repost, time.Now(), nil, nil)

		assertNoError(t, err)
		assertResult(t, result, HandleSkipped)
	})

	t.Run("different messages from different hosts are both inserted", func(t *testing.T) {
		store := &SpyMessageStore{}

		HandleMessage(context.Background(), store, "telegram", "Daniel",
			"Looking for HB players at Heartbeat Bedok, 3 Apr 8pm, $10", time.Now(), nil, nil)
		result, err := HandleMessage(context.Background(), store, "telegram", "Nic",
			"Court let go at cost $16, Hougang CC, 7:30-9:30pm", time.Now(), nil, nil)

		assertNoError(t, err)
		assertResult(t, result, HandleInserted)
		assertCalls(t, store.Calls, []string{"GetRecent", "Insert", "GetRecent", "Insert"})
	})

	t.Run("six messages in rapid succession with two duplicates", func(t *testing.T) {
		store := &SpyMessageStore{}

		cases := []struct {
			sender string
			text   string
			want   HandleResult
		}{
			{"Daniel", "Looking for HB players at Kuo Chuan, 4 Apr 4-6pm, $10", HandleInserted},
			{"Nic", "Court let go at Cost: $16, Hougang CC 5:30-7:30pm", HandleInserted},
			{"Hui Min", "Looking for friendly players, 3 Apr Fri, SBH Expo, HB-LI, $19", HandleInserted},
			{"Daniel", "Looking for HB players at Kuo Chuan, 4 Apr 4-6pm, $10", HandleSkipped},
			{"TB Tay", "Looking for players, 31Mar Farrer Park, 7pm-9pm, HB+, $10/$9", HandleInserted},
			{"Hui Min", "Looking for friendly players, 3 Apr Fri, SBH Expo, HB-LI, $19", HandleSkipped},
		}

		for i, c := range cases {
			result, err := HandleMessage(context.Background(), store, "telegram", c.sender, c.text, time.Now(), nil, nil)
			assertNoError(t, err)
			if result != c.want {
				t.Errorf("message %d (%s): got %d, want %d", i, c.sender, result, c.want)
			}
		}
	})

	t.Run("dedupe always runs before insert", func(t *testing.T) {
		store := &SpyMessageStore{}

		HandleMessage(context.Background(), store, "telegram", "Daniel",
			"Looking for HB players", time.Now(), nil, nil)

		// Verify the operation order: dedup check happens before insert
		if len(store.Calls) < 2 {
			t.Fatal("expected at least 2 calls")
		}
		if store.Calls[0] != "GetRecent" {
			t.Errorf("first call should be GetRecent, got %s", store.Calls[0])
		}
		if store.Calls[1] != "Insert" {
			t.Errorf("second call should be Insert, got %s", store.Calls[1])
		}
	})

	t.Run("insert error returns HandleError", func(t *testing.T) {
		store := &SpyMessageStore{
			InsertErr: fmt.Errorf("database is locked"),
		}

		result, err := HandleMessage(context.Background(), store, "telegram", "Daniel",
			"Looking for HB players", time.Now(), nil, nil)

		assertResult(t, result, HandleError)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("dedupe fetch error still inserts message", func(t *testing.T) {
		store := &SpyMessageStore{
			GetRecentErr: fmt.Errorf("connection reset"),
		}

		result, err := HandleMessage(context.Background(), store, "telegram", "Daniel",
			"Looking for HB players", time.Now(), nil, nil)

		assertNoError(t, err)
		assertResult(t, result, HandleInserted)
		// Should still call both GetRecent (fails) and Insert
		assertCalls(t, store.Calls, []string{"GetRecent", "Insert"})
	})
}

func assertResult(t testing.TB, got, want HandleResult) {
	t.Helper()
	if got != want {
		t.Errorf("got result %d, want %d", got, want)
	}
}

func assertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertCalls(t testing.TB, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %d calls %v, want %d calls %v", len(got), got, len(want), want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("call %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
