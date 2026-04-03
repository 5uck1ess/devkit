package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, ".devkit", "devkit.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateAndGetSession(t *testing.T) {
	db := tempDB(t)

	s := &Session{
		ID:       "abc123def456",
		Workflow: "improve",
		Target:   "src/",
		Metric:   "go test ./...",
		Status:   "running",
	}
	if err := db.CreateSession(s); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := db.GetSession("abc123def456")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Workflow != "improve" {
		t.Errorf("workflow = %q, want improve", got.Workflow)
	}
	if got.Status != "running" {
		t.Errorf("status = %q, want running", got.Status)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	db := tempDB(t)
	_, err := db.GetSession("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestUpdateSessionStatus(t *testing.T) {
	db := tempDB(t)
	s := &Session{ID: "test12345678", Workflow: "improve", Status: "running"}
	db.CreateSession(s)

	if err := db.UpdateSessionStatus("test12345678", "done"); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := db.GetSession("test12345678")
	if got.Status != "done" {
		t.Errorf("status = %q, want done", got.Status)
	}
}

func TestCreateAndGetSteps(t *testing.T) {
	db := tempDB(t)
	db.CreateSession(&Session{ID: "sess12345678", Workflow: "improve", Status: "running"})

	step := &Step{SessionID: "sess12345678", Iteration: 1, Status: "running", AgentName: "claude"}
	if err := db.CreateStep(step); err != nil {
		t.Fatalf("create step: %v", err)
	}
	if step.ID == 0 {
		t.Error("step ID should be set after create")
	}

	step.Status = "kept"
	step.Kept = true
	step.MetricExitCode = 0
	step.CostUSD = 0.05
	if err := db.UpdateStep(step); err != nil {
		t.Fatalf("update step: %v", err)
	}

	steps, err := db.GetSteps("sess12345678")
	if err != nil {
		t.Fatalf("get steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(steps))
	}
	if !steps[0].Kept {
		t.Error("step should be kept")
	}
}

func TestSessionTotalCost(t *testing.T) {
	db := tempDB(t)
	db.CreateSession(&Session{ID: "cost12345678", Workflow: "improve", Status: "running"})

	for i := 1; i <= 3; i++ {
		s := &Step{SessionID: "cost12345678", Iteration: i, Status: "done", AgentName: "claude"}
		db.CreateStep(s)
		s.CostUSD = 0.10
		db.UpdateStep(s)
	}

	cost, err := db.SessionTotalCost("cost12345678")
	if err != nil {
		t.Fatalf("total cost: %v", err)
	}
	if cost < 0.29 || cost > 0.31 {
		t.Errorf("cost = %f, want ~0.30", cost)
	}
}

func TestListSessions(t *testing.T) {
	db := tempDB(t)
	db.CreateSession(&Session{ID: "list12345678", Workflow: "improve", Status: "done"})
	db.CreateSession(&Session{ID: "list87654321", Workflow: "review", Status: "done"})

	sessions, err := db.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}
}

func TestLastIteration(t *testing.T) {
	db := tempDB(t)
	db.CreateSession(&Session{ID: "iter12345678", Workflow: "improve", Status: "running"})

	iter, _ := db.LastIteration("iter12345678")
	if iter != 0 {
		t.Errorf("empty session should have last iter 0, got %d", iter)
	}

	for i := 1; i <= 5; i++ {
		s := &Step{SessionID: "iter12345678", Iteration: i, Status: "done", AgentName: "claude"}
		db.CreateStep(s)
	}

	iter, _ = db.LastIteration("iter12345678")
	if iter != 5 {
		t.Errorf("last iter = %d, want 5", iter)
	}
}

func TestDBDirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, ".devkit")
	OpenDB(filepath.Join(dbDir, "devkit.db"))

	info, err := os.Stat(dbDir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o700 {
		t.Errorf("directory permissions = %o, want 700", perm)
	}
}
