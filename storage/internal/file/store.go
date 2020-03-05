// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"context"
	"encoding/hex"
	"github.com/ystia/yorc/v4/config"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/encryption"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/utils"
)

const indexFileNotExist = uint64(1)

type fileStore struct {
	id         string
	properties config.DynamicMap
	// For locking the locks map
	// (no two goroutines may create a lock for a filename that doesn't have a lock yet).
	locksLock *sync.Mutex
	// For locking file access.
	fileLocks         map[string]*sync.RWMutex
	filenameExtension string
	directory         string
	codec             encoding.Codec
	withCache         bool
	cache             *ristretto.Cache
	withEncryption    bool
	encryptor         *encryption.Encryptor
}

// NewStore returns a new File store
func NewStore(cfg config.Configuration, storeID string, properties config.DynamicMap, withCache, withEncryption bool) (store.Store, error) {
	var err error

	if properties == nil {
		properties = config.DynamicMap{}
	}

	fs := &fileStore{
		id:                storeID,
		properties:        properties,
		codec:             encoding.JSON,
		filenameExtension: "json",
		directory:         properties.GetStringOrDefault("root_dir", path.Join(cfg.WorkingDirectory, "store")),
		locksLock:         new(sync.Mutex),
		fileLocks:         make(map[string]*sync.RWMutex),
		withCache:         withCache,
		withEncryption:    withEncryption,
	}

	// Instantiate cache if necessary
	if withCache {
		if err = fs.initCache(); err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate cache for file store")
		}
	}

	// Instantiate encryptor if necessary
	if withEncryption {
		err = fs.buildEncryptor()
		if err != nil {
			return nil, err
		}
	}

	return fs, nil
}

func (s *fileStore) initCache() error {
	var err error
	// Set Cache config
	numCounters := s.properties.GetInt64("cache_num_counters")
	if numCounters == 0 {
		// 100 000
		numCounters = 1e5
	}
	maxCost := s.properties.GetInt64("cache_max_cost")
	if maxCost == 0 {
		// 10 MB
		maxCost = 1e7
	}
	bufferItems := s.properties.GetInt64("cache_buffer_items")
	if bufferItems == 0 {
		bufferItems = 64
	}

	log.Printf("Instantiate new cache with numCounters:%d, maxCost:%d, bufferItems: %d", numCounters, maxCost, bufferItems)
	s.cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters, // number of keys to track frequency
		MaxCost:     maxCost,     // maximum cost of cache
		BufferItems: bufferItems, // number of keys per Get buffer.
	})
	return err
}

func (s *fileStore) buildEncryptor() error {
	var err error
	// Check a 32-bits passphrase key is provided
	secretKey := s.properties.GetString("passphrase")
	if secretKey == "" {
		return errors.Errorf("Missing passphrase for file store encryption with ID:%q", s.id)
	}
	if len(secretKey) != 32 {
		return errors.Errorf("The provided passphrase for file store encryption with ID:%q must be 32-bits length", s.id)
	}
	s.encryptor, err = encryption.NewEncryptor(hex.EncodeToString([]byte(secretKey)))
	return err
}

// prepareFileLock returns an existing file lock or creates a new one
func (s *fileStore) prepareFileLock(filePath string) *sync.RWMutex {
	s.locksLock.Lock()
	lock, found := s.fileLocks[filePath]
	if !found {
		lock = new(sync.RWMutex)
		s.fileLocks[filePath] = lock
	}
	s.locksLock.Unlock()
	return lock
}

func (s *fileStore) buildFilePath(k string, withExtension bool) string {
	filePath := k
	if withExtension && s.filenameExtension != "" {
		filePath += "." + s.filenameExtension
	}
	return filepath.Join(s.directory, filePath)
}

func (s *fileStore) extractKeyFromFilePath(filePath string, withExtension bool) string {
	key := strings.TrimPrefix(filePath, s.directory+string(os.PathSeparator))
	if !withExtension {
		return key
	}
	key = strings.TrimSuffix(key, "."+s.filenameExtension)
	return key
}

func (s *fileStore) Set(ctx context.Context, k string, v interface{}) error {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	data, err := s.codec.Marshal(v)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal value %+v due to error:%+v", v, err)
	}

	// Prepare file lock.
	filePath := s.buildFilePath(k, true)
	lock := s.prepareFileLock(filePath)

	// File lock and file handling.
	lock.Lock()
	defer lock.Unlock()

	err = os.MkdirAll(path.Dir(filePath), 0700)
	if err != nil {
		return err
	}

	// Copy to cache if necessary
	if s.withCache {
		s.cache.Set(k, data, 1)
	}

	// encrypt if necessary
	if s.withEncryption {
		data, err = s.encryptor.Encrypt(data)
		if err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filePath, data, 0600)
}

func (s *fileStore) SetCollection(ctx context.Context, keyValues []store.KeyValueIn) error {
	if keyValues == nil {
		return nil
	}
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, kv := range keyValues {
		kvItem := kv
		errGroup.Go(func() error {
			return s.Set(ctx, kvItem.Key, kvItem.Value)
		})
	}

	return errGroup.Wait()
}

func (s *fileStore) Get(k string, v interface{}) (bool, error) {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}

	// Check cache first
	if s.withCache {
		value, has := s.cache.Get(k)
		if has {
			data, ok := value.([]byte)
			if ok {
				return true, errors.Wrapf(s.codec.Unmarshal(data, v), "failed to unmarshal data:%q", string(data))
			}
			log.Printf("[WARNING] Failed to cast retrieved value from cache to bytes array for key:%q. Data will be retrieved from store.", k)
		}
	}

	filePath := s.buildFilePath(k, true)
	exist, _, err := s.getValueFromFile(filePath, v)
	return exist, err
}

func (s *fileStore) getValueFromFile(filePath string, v interface{}) (bool, []byte, error) {
	// Prepare file lock.
	lock := s.prepareFileLock(filePath)

	// File lock and file handling.
	lock.RLock()
	// Deferring the unlocking would lead to the unmarshalling being done during the lock, which is bad for performance.
	data, err := ioutil.ReadFile(filePath)
	lock.RUnlock()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, err
	}

	// decrypt if necessary
	if s.withEncryption {
		data, err = s.encryptor.Decrypt(data)
		if err != nil {
			return false, nil, err
		}
	}

	return true, data, errors.Wrapf(s.codec.Unmarshal(data, v), "failed to unmarshal data:%q", string(data))
}

func (s *fileStore) Exist(k string) (bool, error) {
	filePath := s.buildFilePath(k, true)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *fileStore) Keys(k string) ([]string, error) {
	filePath := s.buildFilePath(k, false)

	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	result := make([]string, 0)
	for _, file := range files {
		fileName := file.Name()
		// return the whole key path without specific extension
		result = append(result, path.Join(k, strings.TrimSuffix(fileName, "."+s.filenameExtension)))
	}

	return collections.RemoveDuplicates(result), nil
}

func (s *fileStore) Delete(ctx context.Context, k string, recursive bool) error {
	if err := utils.CheckKey(k); err != nil {
		return err
	}

	// Handle cache
	if s.withCache {
		if err := s.clearCache(ctx, k, recursive); err != nil {
			return err
		}
	}

	var err error

	// Try to delete a single file in all cases
	filePath := s.buildFilePath(k, true)
	// Prepare file lock.
	lock := s.prepareFileLock(filePath)

	// File lock and file handling.
	lock.Lock()
	defer lock.Unlock()

	err = os.Remove(filePath)

	// Try to delete a directory if recursive is true
	if recursive {
		// Remove the whole directory
		err = os.RemoveAll(s.buildFilePath(k, false))
	}

	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *fileStore) clearCache(ctx context.Context, k string, recursive bool) error {
	s.cache.Del(k)

	if !recursive {
		return nil
	}

	// Delete sub-keys
	keys, err := s.Keys(k)
	if err != nil {
		return err
	}
	errGroup, ctx := errgroup.WithContext(ctx)
	for _, key := range keys {
		key := key
		errGroup.Go(func() error {
			return s.clearCache(ctx, key, true)
		})
	}

	return errGroup.Wait()
}

func (s *fileStore) GetLastModifyIndex(k string) (uint64, error) {
	// key can be directory or file, ie with or without extension
	// let's try first without extension as it can be the most current case
	rootPath := s.buildFilePath(k, false)
	fInfo, err := os.Stat(rootPath)

	if err != nil {
		if !os.IsNotExist(err) {
			return 0, errors.Wrapf(err, "failed to get last index for key:%q", k)
		}
		// not a directory, let's try a file with extension
		fp := s.buildFilePath(k, true)
		fInfo, err = os.Stat(fp)
		if err != nil {
			if !os.IsNotExist(err) {
				return 0, errors.Wrapf(err, "failed to get last index for key:%q", k)
			}
			// File not found : the key doesn't exist
			return indexFileNotExist, nil
		}
		return uint64(fInfo.ModTime().UnixNano()), nil
	}

	// For a directory, lastIndex is determined by the max modify index of sub-directories
	var index, lastIndex uint64
	err = filepath.Walk(rootPath, func(pathFile string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			index = uint64(info.ModTime().UnixNano())
			if index > lastIndex {
				lastIndex = index
			}
		}
		return nil
	})
	return lastIndex, err
}

func (s *fileStore) List(ctx context.Context, k string, waitIndex uint64, timeout time.Duration) ([]store.KeyValueOut, uint64, error) {
	if waitIndex == 0 {
		return s.list(k)
	}

	// Default timeout to 5 minutes if not set as param or as config property
	if timeout == 0 {
		timeout = s.properties.GetDuration("blocking_query_default_timeout")
		if timeout == 0 {
			timeout = 5 * time.Minute
		}
	}
	// last index Lookup time interval
	timeAfter := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	var stop bool
	index := uint64(0)
	var err error
	for index <= waitIndex && !stop {
		select {
		case <-ctx.Done():
			log.Debugf("Cancel signal has been received: the store List query is stopped")
			ticker.Stop()
			stop = true
		case <-timeAfter:
			log.Debugf("Timeout has been reached: the store List query is stopped")
			ticker.Stop()
			stop = true
		case <-ticker.C:
			index, err = s.GetLastModifyIndex(k)
			if err != nil {
				return nil, index, err
			}
		}
	}

	return s.list(k)
}

func (s *fileStore) list(k string) ([]store.KeyValueOut, uint64, error) {
	rootPath := s.buildFilePath(k, false)
	fInfo, err := os.Stat(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, indexFileNotExist, nil
		}
		return nil, 0, err
	}
	// Not a directory so nothing to do
	if !fInfo.IsDir() {
		return nil, 0, nil
	}
	var index, lastIndex uint64
	// Fill kv collection recursively for the related directory
	kvs := make([]store.KeyValueOut, 0)

	err = filepath.Walk(rootPath, func(pathFile string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Add kv for a file
		// determine lastIndex for a directory
		if !info.IsDir() {
			var raw []byte
			var value map[string]interface{}
			_, raw, err = s.getValueFromFile(pathFile, &value)
			if err != nil {
				return err
			}

			kv := store.KeyValueOut{
				Key:             s.extractKeyFromFilePath(pathFile, true),
				LastModifyIndex: uint64(info.ModTime().UnixNano()),
				Value:           value,
				RawValue:        raw,
			}
			kvs = append(kvs, kv)
		} else {
			index = uint64(info.ModTime().UnixNano())
			if index > lastIndex {
				lastIndex = index
			}
		}
		return nil
	})

	return kvs, lastIndex, err
}
