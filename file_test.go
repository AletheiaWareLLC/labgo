/*
 * Copyright 2020 Aletheia Ware LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package labgo_test

import (
	"github.com/AletheiaWareLLC/labgo"
	"github.com/AletheiaWareLLC/testinggo"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathToDeltas(t *testing.T) {
	dir, err := ioutil.TempDir("", "foo")
	testinggo.AssertNoError(t, err)
	defer os.RemoveAll(dir)
	file, err := ioutil.TempFile(dir, "bar")
	testinggo.AssertNoError(t, err)
	defer os.Remove(file.Name())
	count, err := file.WriteString("blah")
	testinggo.AssertNoError(t, err)
	if count != 4 {
		t.Fatalf("Could not write to file: expected 4, got '%d'", count)
	}

	var deltas []*labgo.Delta
	testinggo.AssertNoError(t, labgo.PathToDeltas(file.Name(), 10, func(d *labgo.Delta) error {
		deltas = append(deltas, d)
		return nil
	}))
	if len(deltas) != 1 {
		t.Fatalf("Expected 1 Delta, got '%d'", len(deltas))
	}
	delta := deltas[0]
	if delta.Offset != 0 {
		t.Fatalf("Incorrect offset; expected '%d', got '%d'", 0, delta.Offset)
	}
	if len(delta.Remove) != 0 {
		t.Fatalf("Incorrect remove size; expected '%d', got '%d'", 0, len(delta.Remove))
	}
	if string(delta.Remove) != "" {
		t.Fatalf("Incorrect remove; expected '%s', got '%s'", "", string(delta.Remove))
	}
	if len(delta.Add) != 4 {
		t.Fatalf("Incorrect add size; expected '%d', got '%d'", 4, len(delta.Add))
	}
	if string(delta.Add) != "blah" {
		t.Fatalf("Incorrect add; expected '%s', got '%s'", "blah", string(delta.Add))
	}
}

func TestReaderToDeltas(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		want    []*labgo.Delta
	}{
		{
			name:    "Empty",
			initial: "",
			want:    []*labgo.Delta{},
		},
		{
			name:    "Single",
			initial: "foobar",
			want: []*labgo.Delta{
				&labgo.Delta{
					Add: []byte("foobar"),
				},
			},
		},
		{
			name:    "Double",
			initial: "foobarfoobar",
			want: []*labgo.Delta{
				&labgo.Delta{
					Add: []byte("foobarfoob"),
				},
				&labgo.Delta{
					Offset: 10,
					Add:    []byte("ar"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []*labgo.Delta
			testinggo.AssertNoError(t, labgo.ReaderToDeltas(strings.NewReader(tt.initial), 10, func(d *labgo.Delta) error {
				got = append(got, d)
				return nil
			}))
			if len(got) != len(tt.want) {
				t.Fatalf("Wrong number of deltas; expected '%d', got '%d'", len(tt.want), len(got))
			}
			for i, w := range tt.want {
				g := got[i]
				if g.Offset != w.Offset {
					t.Fatalf("Incorrect offset; expected '%d', got '%d'", w.Offset, g.Offset)
				}
				if len(g.Remove) != len(w.Remove) {
					t.Fatalf("Incorrect remove size; expected '%d', got '%d'", len(w.Remove), len(g.Remove))
				}
				if string(g.Remove) != string(w.Remove) {
					t.Fatalf("Incorrect remove; expected '%s', got '%s'", string(w.Remove), string(g.Remove))
				}
				if len(g.Add) != len(w.Add) {
					t.Fatalf("Incorrect add size; expected '%d', got '%d'", len(w.Add), len(g.Add))
				}
				if string(g.Add) != string(w.Add) {
					t.Fatalf("Incorrect add; expected '%s', got '%s'", string(w.Add), string(g.Add))
				}
			}
		})
	}
}

func TestDeltaToBuffer(t *testing.T) {
	tests := []struct {
		name    string
		delta   *labgo.Delta
		initial string
		want    string
	}{
		{
			name:    "Empty",
			delta:   &labgo.Delta{},
			initial: "foobar",
			want:    "foobar",
		},
		{
			name: "Remove",
			delta: &labgo.Delta{
				Remove: []byte("foo"),
			},
			initial: "foobar",
			want:    "bar",
		},
		{
			name: "RemoveAll",
			delta: &labgo.Delta{
				Remove: []byte("foobar"),
			},
			initial: "foobar",
			want:    "",
		},
		{
			name: "Append",
			delta: &labgo.Delta{
				Offset: 6,
				Add:    []byte("blah"),
			},
			initial: "foobar",
			want:    "foobarblah",
		},
		{
			name: "Insert",
			delta: &labgo.Delta{
				Offset: 3,
				Add:    []byte("blah"),
			},
			initial: "foobar",
			want:    "fooblahbar",
		},
		{
			name: "Replace",
			delta: &labgo.Delta{
				Offset: 3,
				Remove: []byte("bar"),
				Add:    []byte("blah"),
			},
			initial: "foobar",
			want:    "fooblah",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(labgo.DeltaToBuffer(tt.delta, []byte(tt.initial)))
			if got != tt.want {
				t.Fatalf("Incorrect buffer; expected '%s', got '%s'", tt.want, got)
			}
		})
	}
}

func TestDeltaToPath(t *testing.T) {
	tests := []struct {
		name    string
		delta   *labgo.Delta
		initial string
		want    string
	}{
		{
			name:    "Empty",
			delta:   &labgo.Delta{},
			initial: "foobar",
			want:    "foobar",
		},
		{
			name: "Remove",
			delta: &labgo.Delta{
				Remove: []byte("foo"),
			},
			initial: "foobar",
			want:    "bar",
		},
		{
			name: "RemoveAll",
			delta: &labgo.Delta{
				Remove: []byte("foobar"),
			},
			initial: "foobar",
			want:    "",
		},
		{
			name: "Append",
			delta: &labgo.Delta{
				Offset: 6,
				Add:    []byte("blah"),
			},
			initial: "foobar",
			want:    "foobarblah",
		},
		{
			name: "Insert",
			delta: &labgo.Delta{
				Offset: 3,
				Add:    []byte("blah"),
			},
			initial: "foobar",
			want:    "fooblahbar",
		},
		{
			name: "Replace",
			delta: &labgo.Delta{
				Offset: 3,
				Remove: []byte("bar"),
				Add:    []byte("blah"),
			},
			initial: "foobar",
			want:    "fooblah",
		},
	}
	dir, err := ioutil.TempDir("", "foo")
	testinggo.AssertNoError(t, err)
	defer os.RemoveAll(dir)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Println(tt.name, tt.delta)
			file, err := os.Create(filepath.Join(dir, "bar"))
			testinggo.AssertNoError(t, err)
			testinggo.AssertNoError(t, ioutil.WriteFile(file.Name(), []byte(tt.initial), 0666))
			testinggo.AssertNoError(t, labgo.DeltaToPath(tt.delta, file.Name()))
			data, err := ioutil.ReadFile(file.Name())
			testinggo.AssertNoError(t, err)
			got := string(data)
			if got != tt.want {
				t.Fatalf("Incorrect file; expected '%s', got '%s'", tt.want, got)
			}
		})
	}
}
