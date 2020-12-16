// Copyright 2020 The Starship Troopers Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"time"
)

var ttlForever = time.Duration(0)

type Storage interface {
	Get(uid string) (payload interface{}, createdAt time.Time, ttl time.Duration, err error)
	Put(uid string, payload interface{}, ttl *time.Duration) error
}

type DataStorage struct {
	mds Storage
	pds Storage
}

func NewDataStorage(memoryStorage Storage, fsStorage Storage) *DataStorage {
	return &DataStorage{
		memoryStorage,
		fsStorage,
	}
}

func (d *DataStorage) Get(uid string) (payload interface{}, createdAt time.Time, ttl time.Duration, err error) {
	return d.mds.Get(uid)
}

//ttl isn't supported and ignored here
func (d *DataStorage) Put(uid string, payload interface{}, ttl *time.Duration) error {
	if ttl != nil && *ttl == ttlForever {
		if err := d.pds.Put(uid, payload, nil); err != nil {
			return fmt.Errorf("can't store data into the persistent storage: %v", err)
		}
	}
	if err := d.mds.Put(uid, payload, ttl); err != nil {
		return fmt.Errorf("can't store data into the memory storage: %v", err)
	}
	return nil
}
