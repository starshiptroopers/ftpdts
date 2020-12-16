// Copyright 2020 The Starship Troopers Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type UIDGenerator interface {
	Validate(string) (string, error)
}

type FsDataStorage struct {
	path         string
	uidGenerator UIDGenerator
}

func NewFsDataStorage(path string, uid UIDGenerator) *FsDataStorage {
	return &FsDataStorage{
		path,
		uid,
	}
}

func (t *FsDataStorage) Get(uid string) (payload interface{}, createdAt time.Time, ttl time.Duration, err error) {

	fPath, err := t.filePath(uid)
	if err != nil {
		return
	}

	info, err := os.Stat(fPath)
	if err != nil {
		err = fmt.Errorf("can't stat the file: %v\n", err)
		return
	}
	createdAt = info.ModTime()

	f, err := os.OpenFile(fPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("can't open the file: %v\n", err)
		return
	}

	defer f.Close()

	b := bytes.NewBuffer(make([]byte, 0))
	_, err = b.ReadFrom(f)
	if err != nil {
		err = fmt.Errorf("can't read the file: %v\n", err)
		return
	}
	err = json.Unmarshal(b.Bytes(), &payload)
	if err != nil {
		err = fmt.Errorf("can't parse json from file: %v\n", err)
		return
	}

	return
}

//ttl isn't supported and ignored here
func (t *FsDataStorage) Put(uid string, payload interface{}, ttl *time.Duration) error {

	fPath, err := t.filePath(uid)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(fPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("can't open the file: %v\n", err)
	}

	defer f.Close()

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("wrong json data: %v\n", err)
	}

	_, err = f.Write(b)

	if err != nil {
		return fmt.Errorf("Can't write the data to file: %v\n", err)
	}

	return nil
}

func (t *FsDataStorage) Pass(callback func(uid string, createdAt time.Time, data interface{})) error {
	files, err := ioutil.ReadDir(t.path)
	if err != nil {
		return fmt.Errorf("Can't read the path: %v\n", err)
	}

	for _, f := range files {
		p, c, _, err := t.Get(f.Name())
		if err != nil {
			continue
		}
		callback(f.Name(), c, p)
	}
	return nil
}

func (t *FsDataStorage) filePath(uid string) (path string, err error) {
	u, err := t.uidGenerator.Validate(uid)
	if err != nil || u != uid {
		err = errors.New("wrong uid")
		return
	}

	path, err = filepath.Abs(t.path + string(filepath.Separator) + u)
	if err != nil {
		err = errors.New("wrong path")
	}
	return
}
