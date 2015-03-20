// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func remove(index int, list ...interface{}) []interface{} {
	var (
		left  []interface{}
		right []interface{}
	)

	switch {
	case index < 0 || index > len(list):
		panic("index out of bounds")
	case index+1 < len(list):
		right = list[index+1:]
		fallthrough
	default:
		left = list[:index]
	}

	return append(left, right...)
}

var editors = []string{"vim", "vi", "nano"}

func findEditor(editor string) (string, error) {
	if editor != "" {
		path, err := exec.LookPath(editor)
		if err != nil {
			return "", fmt.Errorf("editor (%s) not found: %s", editor, err)
		}
		return path, nil
	}
	for _, e := range editors {
		path, err := exec.LookPath(e)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no editor found")
}

func openEditor(data []byte, name, editor string) (reader io.Reader, err error) {
	if terminal.IsTerminal(syscall.Stdin) {
		editor, err := findEditor(editor)
		if err != nil {
			return nil, err
		}

		f, err := ioutil.TempFile("", name+"_")
		if err != nil {
			return nil, fmt.Errorf("could not open tempfile: %s", err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

		if _, err := f.Write(data); err != nil {
			return nil, fmt.Errorf("could not write tempfile: %s", err)
		}

		e := exec.Command(editor, f.Name())
		e.Stdin = os.Stdin
		e.Stdout = os.Stdout
		e.Stderr = os.Stderr

		if err := e.Run(); err != nil {
			return nil, fmt.Errorf("received error from editor: %s (%s)", err, editor)
		}
		if _, err := f.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("could not seek file: %s", err)
		}

		data, err := ioutil.ReadFile(f.Name())
		if err != nil {
			return nil, fmt.Errorf("could not read file: %s", err)
		}

		reader = bytes.NewReader(data)
	} else {
		if _, err := os.Stdout.Write(data); err != nil {
			return nil, fmt.Errorf("could not write to stdout: %s", err)
		}
		reader = os.Stdin
	}

	return reader, nil
}
