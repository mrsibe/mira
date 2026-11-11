package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrsibe/goat/ai"
)

func TestProjectsAndConversationAssignmentPersist(t *testing.T) {
	dir := t.TempDir()
	conversations, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := conversations.CreateProject("Product")
	if err != nil {
		t.Fatal(err)
	}
	second, err := conversations.CreateProject("Research")
	if err != nil {
		t.Fatal(err)
	}
	projects := conversations.ListProjects()
	if len(projects) != 2 || projects[0].ID != first.ID || projects[1].ID != second.ID {
		t.Fatalf("projects = %+v", projects)
	}

	unfiled, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	filed, err := conversations.CreateInProject(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unfiled.ProjectID != "" || filed.ProjectID != first.ID {
		t.Fatalf("created conversations = %+v, %+v", unfiled, filed)
	}
	if err := conversations.Append(filed.ID, ai.TextMessage(ai.RoleUser, "hello")); err != nil {
		t.Fatal(err)
	}
	if err := conversations.MoveConversation(filed.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := conversations.MoveConversation(unfiled.ID, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := conversations.MoveConversation(unfiled.ID, ""); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(dir, "index.json")); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]Conversation)
	for _, conversation := range reopened.List() {
		byID[conversation.ID] = conversation
	}
	if byID[filed.ID].ProjectID != second.ID {
		t.Fatalf("filed project after rebuild = %q", byID[filed.ID].ProjectID)
	}
	if byID[unfiled.ID].ProjectID != "" {
		t.Fatalf("unfiled project after rebuild = %q", byID[unfiled.ID].ProjectID)
	}
}

func TestProjectNamesAreValidated(t *testing.T) {
	conversations, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := conversations.CreateProject("  Product  ")
	if err != nil {
		t.Fatal(err)
	}
	if project.Name != "Product" {
		t.Fatalf("project name = %q", project.Name)
	}
	for _, name := range []string{"", "未分类", "product", "PRODUCT"} {
		if _, err := conversations.CreateProject(name); err == nil {
			t.Fatalf("CreateProject(%q) succeeded", name)
		}
	}
	other, err := conversations.CreateProject("Other")
	if err != nil {
		t.Fatal(err)
	}
	if err := conversations.RenameProject(other.ID, "product"); err == nil {
		t.Fatal("RenameProject accepted a duplicate name")
	}
	if err := conversations.RenameProject(project.ID, "Product renamed"); err != nil {
		t.Fatal(err)
	}
}

func TestMoveConversationRejectsMissingProject(t *testing.T) {
	conversations, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	if err := conversations.MoveConversation(conversation.ID, "missing"); err == nil ||
		!strings.Contains(err.Error(), "项目不存在") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeleteProjectRemovesItsConversations(t *testing.T) {
	dir := t.TempDir()
	conversations, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	project, err := conversations.CreateProject("Delete me")
	if err != nil {
		t.Fatal(err)
	}
	keep, err := conversations.Create()
	if err != nil {
		t.Fatal(err)
	}
	remove, err := conversations.CreateInProject(project.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := conversations.DeleteProject(project.ID); err != nil {
		t.Fatal(err)
	}
	if len(conversations.ListProjects()) != 0 {
		t.Fatal("deleted project is still listed")
	}
	list := conversations.List()
	if len(list) != 1 || list[0].ID != keep.ID {
		t.Fatalf("conversations = %+v", list)
	}
	if _, err := os.Stat(filepath.Join(dir, remove.ID+".jsonl")); !os.IsNotExist(err) {
		t.Fatalf("removed conversation file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, keep.ID+".jsonl")); err != nil {
		t.Fatalf("kept conversation file: %v", err)
	}
}
