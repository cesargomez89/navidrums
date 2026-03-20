package store

import (
	"testing"
)

func TestProvidersRepo_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id1 := repo.Create("http://provider1.example", "Provider 1")
	if id1 == 0 {
		t.Fatal("Expected non-zero ID")
	}

	id2 := repo.Create("http://provider2.example", "Provider 2")
	if id2 == 0 {
		t.Fatal("Expected non-zero ID")
	}

	if id2 <= id1 {
		t.Errorf("Expected id2 > id1, got id1=%d, id2=%d", id1, id2)
	}

	p1, err := repo.GetByPosition(0)
	if err != nil {
		t.Fatalf("GetByPosition failed: %v", err)
	}
	if p1.ID != id1 {
		t.Errorf("Position 0 should have id %d, got %d", id1, p1.ID)
	}

	p2, err := repo.GetByPosition(1)
	if err != nil {
		t.Fatalf("GetByPosition failed: %v", err)
	}
	if p2.ID != id2 {
		t.Errorf("Position 1 should have id %d, got %d", id2, p2.ID)
	}
}

func TestProvidersRepo_CreateDuplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id1 := repo.Create("http://dup.example", "First")
	if id1 == 0 {
		t.Fatal("Expected non-zero ID")
	}

	id2 := repo.Create("http://dup.example", "Second")
	if id2 != 0 {
		t.Errorf("Expected id=0 for duplicate, got %d", id2)
	}

	list, err := repo.ListOrdered()
	if err != nil {
		t.Fatalf("ListOrdered failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 provider after duplicate insert, got %d", len(list))
	}
}

func TestProvidersRepo_ListOrdered(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	repo.Create("http://c.example", "C")
	repo.Create("http://a.example", "A")
	repo.Create("http://b.example", "B")

	providers, err := repo.ListOrdered()
	if err != nil {
		t.Fatalf("ListOrdered failed: %v", err)
	}

	if len(providers) != 3 {
		t.Fatalf("Expected 3 providers, got %d", len(providers))
	}

	if providers[0].Position != 0 || providers[1].Position != 1 || providers[2].Position != 2 {
		t.Errorf("Positions not in order: %d, %d, %d",
			providers[0].Position, providers[1].Position, providers[2].Position)
	}

	if providers[0].Name != "C" || providers[1].Name != "A" || providers[2].Name != "B" {
		t.Errorf("Unexpected order: %s, %s, %s",
			providers[0].Name, providers[1].Name, providers[2].Name)
	}
}

func TestProvidersRepo_ListOrdered_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	providers, err := repo.ListOrdered()
	if err != nil {
		t.Fatalf("ListOrdered failed: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}
}

func TestProvidersRepo_GetByPosition(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id := repo.Create("http://test.example", "Test")
	if id == 0 {
		t.Fatal("Expected non-zero ID")
	}

	p, err := repo.GetByPosition(0)
	if err != nil {
		t.Fatalf("GetByPosition failed: %v", err)
	}
	if p.ID != id {
		t.Errorf("ID = %d, want %d", p.ID, id)
	}
}

func TestProvidersRepo_GetByPosition_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	_, err := repo.GetByPosition(999)
	if err == nil {
		t.Error("Expected error for non-existent position")
	}
}

func TestProvidersRepo_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id := repo.Create("http://old.example", "Old Name")
	if id == 0 {
		t.Fatal("Expected non-zero ID")
	}

	err := repo.Update(id, "http://new.example", "New Name")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	p, err := repo.GetByPosition(0)
	if err != nil {
		t.Fatalf("GetByPosition failed: %v", err)
	}
	if p.URL != "http://new.example" {
		t.Errorf("URL = %q, want %q", p.URL, "http://new.example")
	}
	if p.Name != "New Name" {
		t.Errorf("Name = %q, want %q", p.Name, "New Name")
	}
}

func TestProvidersRepo_Update_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	err := repo.Update(99999, "http://test.example", "Test")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestProvidersRepo_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id := repo.Create("http://delete.example", "Delete Me")
	if id == 0 {
		t.Fatal("Expected non-zero ID")
	}

	err := repo.Delete(id)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByPosition(0)
	if err == nil {
		t.Error("Expected error after delete")
	}

	list, err := repo.ListOrdered()
	if err != nil {
		t.Fatalf("ListOrdered failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("Expected 0 providers after delete, got %d", len(list))
	}
}

func TestProvidersRepo_Delete_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	err := repo.Delete(99999)
	if err != nil {
		t.Errorf("Delete non-existent should not error: %v", err)
	}
}

func TestProvidersRepo_Exists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id := repo.Create("http://exists.example", "Exists")
	if id == 0 {
		t.Fatal("Expected non-zero ID")
	}

	if !repo.Exists("http://exists.example") {
		t.Error("Expected Exists to return true")
	}

	if repo.Exists("http://nonexistent.example") {
		t.Error("Expected Exists to return false for non-existent")
	}
}

func TestProvidersRepo_Reorder(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	id1 := repo.Create("http://a.example", "A")
	id2 := repo.Create("http://b.example", "B")
	id3 := repo.Create("http://c.example", "C")

	err := repo.Reorder([]int64{id3, id1, id2})
	if err != nil {
		t.Fatalf("Reorder failed: %v", err)
	}

	providers, err := repo.ListOrdered()
	if err != nil {
		t.Fatalf("ListOrdered failed: %v", err)
	}

	if len(providers) != 3 {
		t.Fatalf("Expected 3 providers, got %d", len(providers))
	}

	if providers[0].ID != id3 || providers[0].Position != 0 {
		t.Errorf("Position 0: expected id=%d, got id=%d, pos=%d", id3, providers[0].ID, providers[0].Position)
	}
	if providers[1].ID != id1 || providers[1].Position != 1 {
		t.Errorf("Position 1: expected id=%d, got id=%d, pos=%d", id1, providers[1].ID, providers[1].Position)
	}
	if providers[2].ID != id2 || providers[2].Position != 2 {
		t.Errorf("Position 2: expected id=%d, got id=%d, pos=%d", id2, providers[2].ID, providers[2].Position)
	}
}

func TestProvidersRepo_Reorder_PartialUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	repo.Create("http://a.example", "A")
	id2 := repo.Create("http://b.example", "B")
	repo.Create("http://c.example", "C")

	err := repo.Reorder([]int64{id2})
	if err != nil {
		t.Fatalf("Reorder failed: %v", err)
	}

	providers, err := repo.ListOrdered()
	if err != nil {
		t.Fatalf("ListOrdered failed: %v", err)
	}

	if providers[0].ID != id2 {
		t.Errorf("Position 0 should contain id2=%d, got id=%d", id2, providers[0].ID)
	}

	if providers[0].Position != 0 {
		t.Errorf("B position = %d, want 0", providers[0].Position)
	}
}

func TestProvidersRepo_Reorder_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProvidersRepo(db)

	err := repo.Reorder([]int64{})
	if err != nil {
		t.Fatalf("Reorder with empty slice should not error: %v", err)
	}
}
