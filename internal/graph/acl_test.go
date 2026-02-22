package graph

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPermissionIsExternal(t *testing.T) {
	p := Permission{
		GrantedToV2: &Principal{
			User: &GraphUser{Email: "partner@acme.com"},
		},
	}
	if !p.IsExternal("company.com") {
		t.Error("expected external for partner@acme.com vs company.com")
	}
	if p.IsExternal("acme.com") {
		t.Error("should not be external for acme.com")
	}
}

func TestPermissionIsInherited(t *testing.T) {
	inherited := Permission{
		InheritedFrom: &struct {
			ID string `json:"id"`
		}{ID: "parent-1"},
	}
	if !inherited.IsInherited() {
		t.Error("expected inherited")
	}

	unique := Permission{}
	if unique.IsInherited() {
		t.Error("expected not inherited")
	}
}

func TestPermissionIsAnonymousLink(t *testing.T) {
	anon := Permission{
		Link: &PermLink{Scope: "anonymous", Type: "edit"},
	}
	if !anon.IsAnonymousLink() {
		t.Error("expected anonymous link")
	}

	org := Permission{
		Link: &PermLink{Scope: "organization", Type: "view"},
	}
	if org.IsAnonymousLink() {
		t.Error("should not be anonymous link")
	}

	noLink := Permission{}
	if noLink.IsAnonymousLink() {
		t.Error("nil link should not be anonymous")
	}
}

func TestFindExternalShares(t *testing.T) {
	report := &ACLReport{
		Entries: []ACLEntry{
			{Path: "doc1.docx", ExternalUsers: []string{"ext@other.com"}},
			{Path: "doc2.docx"},
			{Path: "doc3.xlsx", ExternalUsers: []string{"partner@acme.com"}},
		},
	}

	result := FindExternalShares(report)
	if len(result) != 2 {
		t.Errorf("expected 2 external shares, got %d", len(result))
	}
}

func TestFindBrokenInheritance(t *testing.T) {
	report := &ACLReport{
		Entries: []ACLEntry{
			{Path: "doc1.docx", HasUniquePermissions: true},
			{Path: "doc2.docx", HasUniquePermissions: false},
			{Path: "doc3.xlsx", HasUniquePermissions: true},
		},
	}

	result := FindBrokenInheritance(report)
	if len(result) != 2 {
		t.Errorf("expected 2 broken inheritance, got %d", len(result))
	}
}

func TestACLReportAggregation(t *testing.T) {
	report := &ACLReport{
		TotalFiles:        5,
		ExternalShares:    2,
		BrokenInheritance: 1,
		AnonymousLinks:    3,
		Entries: []ACLEntry{
			{Path: "a.docx", ExternalUsers: []string{"ext@other.com"}, HasUniquePermissions: true},
			{Path: "b.docx", ExternalUsers: []string{"ext2@other.com"}},
			{Path: "c.docx"},
		},
	}

	if report.TotalFiles != 5 {
		t.Errorf("TotalFiles = %d", report.TotalFiles)
	}
	if report.ExternalShares != 2 {
		t.Errorf("ExternalShares = %d", report.ExternalShares)
	}
	if report.BrokenInheritance != 1 {
		t.Errorf("BrokenInheritance = %d", report.BrokenInheritance)
	}
	if CountAnonymousLinks(report) != 3 {
		t.Errorf("AnonymousLinks = %d", CountAnonymousLinks(report))
	}
}

func TestEmptySiteNoFindings(t *testing.T) {
	report := &ACLReport{
		Entries: []ACLEntry{},
	}

	external := FindExternalShares(report)
	broken := FindBrokenInheritance(report)

	if len(external) != 0 {
		t.Error("expected no external shares")
	}
	if len(broken) != 0 {
		t.Error("expected no broken inheritance")
	}
}

func TestNewACL(t *testing.T) {
	client := &http.Client{}
	a := NewACL(client, "company.com")
	if a == nil {
		t.Fatal("expected non-nil ACL")
	}
	if a.OrgDomain != "company.com" {
		t.Errorf("OrgDomain = %q", a.OrgDomain)
	}
}

func TestPermissionGetEmail(t *testing.T) {
	p1 := Permission{
		GrantedToV2: &Principal{User: &GraphUser{Email: "alice@test.com"}},
	}
	if p1.GetEmail() != "alice@test.com" {
		t.Errorf("GetEmail = %q", p1.GetEmail())
	}

	p2 := Permission{
		GrantedTo: &Principal{User: &GraphUser{Email: "bob@test.com"}},
	}
	if p2.GetEmail() != "bob@test.com" {
		t.Errorf("GetEmail = %q", p2.GetEmail())
	}

	p3 := Permission{}
	if p3.GetEmail() != "" {
		t.Errorf("expected empty email, got %q", p3.GetEmail())
	}
}

func TestACLReportJSON(t *testing.T) {
	report := ACLReport{
		Site:           "test-site",
		TotalFiles:     10,
		ExternalShares: 2,
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	var decoded ACLReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Site != "test-site" {
		t.Errorf("Site = %q", decoded.Site)
	}
	if decoded.TotalFiles != 10 {
		t.Errorf("TotalFiles = %d", decoded.TotalFiles)
	}
}
