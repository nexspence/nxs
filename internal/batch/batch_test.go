package batch_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/nexspence/nxs/internal/batch"
)

func TestWalk_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("x"), 0o644)

	jobs, err := batch.Walk(f, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].LocalPath != f || jobs[0].RelPath != "a.txt" {
		t.Errorf("unexpected: %+v", jobs)
	}
}

func TestWalk_Directory(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("y"), 0o644)

	jobs, err := batch.Walk(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	rels := []string{}
	for _, j := range jobs {
		rels = append(rels, j.RelPath)
	}
	sort.Strings(rels)
	if len(rels) != 2 || rels[0] != "a.txt" || rels[1] != "sub/b.txt" {
		t.Errorf("unexpected rels: %v", rels)
	}
}

func TestWalk_Glob(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "build"), 0o755)
	os.WriteFile(filepath.Join(dir, "build", "app.jar"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "build", "app.txt"), []byte("y"), 0o644)

	jobs, err := batch.Walk(filepath.Join(dir, "build", "**", "*.jar"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || filepath.Base(jobs[0].LocalPath) != "app.jar" {
		t.Errorf("glob mismatch: %+v", jobs)
	}
}

func TestRunPool_AllSucceed(t *testing.T) {
	jobs := []batch.Job{{RelPath: "1"}, {RelPath: "2"}, {RelPath: "3"}}
	var count int32
	res := batch.RunPool(jobs, 2, false, func(j batch.Job) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	if res.OK != 3 || len(res.Failed) != 0 {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestRunPool_ContinueOnError(t *testing.T) {
	jobs := []batch.Job{{RelPath: "1"}, {RelPath: "2"}, {RelPath: "3"}}
	res := batch.RunPool(jobs, 2, true, func(j batch.Job) error {
		if j.RelPath == "2" {
			return fmt.Errorf("boom")
		}
		return nil
	})
	if res.OK != 2 || len(res.Failed) != 1 {
		t.Errorf("expected 2 ok / 1 failed, got %+v", res)
	}
}
