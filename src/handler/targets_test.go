package handler

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestUUIDFromStringArray_Deterministic(t *testing.T) {
	targets := []string{"10.0.0.1:9100", "10.0.0.2:9100"}
	a := UUIDFromStringArray(targets)
	b := UUIDFromStringArray(targets)
	if a != b {
		t.Fatalf("expected same input to produce the same UUID, got %s and %s", a, b)
	}
}

func TestUUIDFromStringArray_CommaInAddressDoesNotCollide(t *testing.T) {
	a := UUIDFromStringArray([]string{"a,b"})
	b := UUIDFromStringArray([]string{"a", "b"})
	if a == b {
		t.Fatalf("expected [\"a,b\"] and [\"a\",\"b\"] to hash to different UUIDs, both were %s", a)
	}
}

func TestUUIDFromStringArray_DifferentInputsDiffer(t *testing.T) {
	a := UUIDFromStringArray([]string{"10.0.0.1:9100"})
	b := UUIDFromStringArray([]string{"10.0.0.2:9100"})
	if a == b {
		t.Fatalf("expected different targets to produce different UUIDs, both were %s", a)
	}
}

func TestInsert_DefaultsGroupAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	target := StaticConfig{Targets: []string{"10.0.0.1:9100"}, Labels: map[string]string{"job": "node"}}

	id1, err := c.Insert(target, ctx, con, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := c.Retrieve(id1, ctx, con)
	if err != nil {
		t.Fatalf("unexpected error retrieving inserted target: %v", err)
	}
	if stored.Group != "inventor-default" {
		t.Errorf("expected default group %q, got %q", "inventor-default", stored.Group)
	}

	id2, err := c.Insert(target, ctx, con, 60)
	if err != nil {
		t.Fatalf("unexpected error on second insert: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected re-registering the same targets to be idempotent, got ids %s and %s", id1, id2)
	}
}

func TestDelete_ReturnsNotFoundForMissingID(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	_, err := c.Delete(uuid.New(), ctx, con)
	if err != ErrIDNotFound {
		t.Fatalf("expected ErrIDNotFound for a missing id, got %v", err)
	}
}

func TestDelete_RemovesExistingTarget(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	id, err := c.Insert(StaticConfig{Targets: []string{"10.0.0.1:9100"}}, ctx, con, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ok, err := c.Delete(id, ctx, con)
	if err != nil || !ok {
		t.Fatalf("expected successful delete, got ok=%v err=%v", ok, err)
	}

	if _, err := c.Retrieve(id, ctx, con); err != ErrIDNotFound {
		t.Fatalf("expected target to be gone after delete, got err=%v", err)
	}
}

func TestRetrieve_UnmarshalFailureOnCorruptData(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	id := uuid.New()
	if err := con.Set(ctx, id.String(), "not-json", 0).Err(); err != nil {
		t.Fatalf("failed to seed corrupt value: %v", err)
	}

	if _, err := c.Retrieve(id, ctx, con); err != ErrUnmarshalFailed {
		t.Fatalf("expected ErrUnmarshalFailed for corrupt data, got %v", err)
	}
}

func TestScan_ReturnsAllInsertedTargets(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	first, err := c.Insert(StaticConfig{Targets: []string{"10.0.0.1:9100"}, Group: "a"}, ctx, con, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := c.Insert(StaticConfig{Targets: []string{"10.0.0.2:9100"}, Group: "b"}, ctx, con, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := c.Scan(ctx, con)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if _, ok := result.Items[first]; !ok {
		t.Errorf("expected scan to contain first inserted target %s", first)
	}
	if _, ok := result.Items[second]; !ok {
		t.Errorf("expected scan to contain second inserted target %s", second)
	}
}

func TestScan_SkipsCorruptEntryWithoutFailingWholeScan(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	good, err := c.Insert(StaticConfig{Targets: []string{"10.0.0.1:9100"}}, ctx, con, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bad := uuid.New()
	if err := con.Set(ctx, bad.String(), "not-json", 0).Err(); err != nil {
		t.Fatalf("failed to seed corrupt value: %v", err)
	}

	result, err := c.Scan(ctx, con)
	if err != nil {
		t.Fatalf("unexpected error from scan: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected scan to skip the corrupt entry and return 1 item, got %d", len(result.Items))
	}
	if _, ok := result.Items[good]; !ok {
		t.Errorf("expected scan to still contain the valid target %s", good)
	}
}

func TestScan_EmptyDatabase(t *testing.T) {
	ctx := context.Background()
	con := newTestClient(t)
	c := &SDTargets{Items: make(map[uuid.UUID]StaticConfig)}

	result, err := c.Scan(ctx, con)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no items in an empty database, got %d", len(result.Items))
	}
}
