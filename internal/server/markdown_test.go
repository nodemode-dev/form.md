package server_test

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"
)

const sample = "# Quarterly plan\n\nIntro paragraph with **bold** and `code`.\n\n## Goals\n\n- first goal\n- second goal\n\n1. step one\n2. step two\n\n| Name | Count |\n| --- | --- |\n| apples | 3 |\n| pears | 5 |\n\n```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n\n### Notes\n\nClosing line.\n"

func requireStructure(t *testing.T, md string) {
	t.Helper()
	for _, want := range []string{
		"## Goals", "### Notes",
		"- first goal", "- second goal", "1. step one", "2. step two",
		"| Name | Count |", "| --- | --- |", "| apples | 3 |", "| pears | 5 |",
		"```go", "func main() {", "println(\"hi\")", "**bold**", "`code`", "Closing line.",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("exported Markdown lost %q:\n%s", want, md)
		}
	}
	if strings.Index(md, "```go") > strings.Index(md, "### Notes") || strings.Index(md, "| Name |") > strings.Index(md, "```go") {
		t.Errorf("blocks changed order:\n%s", md)
	}
}

func TestMarkdownWriteThenReadKeepsStructure(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	page := owner.createPage(owner.workspaceID(), "Plan")
	owner.putMarkdown(page, sample)
	first := serverMarkdown(owner, page)
	requireStructure(t, first)
	if !strings.HasPrefix(first, "# Quarterly plan\n") {
		t.Fatalf("the top heading was lost:\n%s", first)
	}
	owner.putMarkdown(page, first)
	second := serverMarkdown(owner, page)
	if second != first {
		t.Fatalf("a second round trip changed the document:\n--- first\n%s\n--- second\n%s", first, second)
	}
}

func upload(c *client, name, format string, content []byte, fields map[string]string) reply {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.WriteField("format", format)
	fw, _ := mw.CreateFormFile("file", name)
	_, _ = fw.Write(content)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", c.e.ts.URL+"/api/imports", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return reply{Status: resp.StatusCode, Body: b, Header: resp.Header}
}

func TestMarkdownFileImportThenExportKeepsStructure(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	ws := owner.workspaceID()
	r := upload(owner, "Quarterly plan.md", "markdown", []byte(sample), map[string]string{"workspace_id": ws})
	if r.Status != http.StatusAccepted {
		t.Fatalf("import refused: %d %s", r.Status, r.Body)
	}
	job := r.JSON(t)["job"].(map[string]any)
	id := job["id"].(string)
	var roots []any
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		got := owner.mustJSON("GET", "/api/imports/"+id, nil, http.StatusOK)["job"].(map[string]any)
		if got["state"] == "failed" {
			t.Fatalf("import failed: %v", got)
		}
		if got["state"] == "done" {
			roots, _ = got["root_ids"].([]any)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(roots) != 1 {
		t.Fatalf("the import produced %d pages, want 1", len(roots))
	}
	page := roots[0].(string)
	requireStructure(t, serverMarkdown(owner, page))

	ex := owner.do("GET", "/api/pages/"+page+"/export?format=markdown", nil)
	if ex.Status != http.StatusOK {
		t.Fatalf("export: %d %s", ex.Status, ex.Body)
	}
	zr, err := zip.NewReader(bytes.NewReader(ex.Body), int64(len(ex.Body)))
	if err != nil {
		t.Fatal(err)
	}
	var md string
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".md") {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			md = string(b)
		}
	}
	if md == "" {
		t.Fatalf("the export zip has no Markdown file")
	}
	requireStructure(t, md)
}
