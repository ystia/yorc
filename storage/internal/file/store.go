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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/storage/encoding"
	"github.com/ystia/yorc/v4/storage/encryption"
	"github.com/ystia/yorc/v4/storage/store"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/storage/utils"
)

type fileStore struct {
	id string
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
	cc                *api.Client
}

// NewStore returns a new File store
func NewStore(cfg config.Configuration, storeID, rootDir string, withCache, withEncryption bool) (store.Store, error) {
	var err error
	fs := &fileStore{
		id:                storeID,
		codec:             encoding.JSON,
		filenameExtension: "json",
		directory:         rootDir,
		locksLock:         new(sync.Mutex),
		fileLocks:         make(map[string]*sync.RWMutex),
		withCache:         withCache,
		withEncryption:    withEncryption,
	}
	// Use consul to store shared encryption key btw yorc instances
	fs.cc, err = cfg.GetConsulClient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get consul client for new file store")
	}

	// Instantiate cache if necessary
	if withCache {
		fs.cache, err = ristretto.NewCache(&ristretto.Config{
			NumCounters: 1e7,     // number of keys to track frequency of (10M).
			MaxCost:     1 << 30, // maximum cost of cache (1GB).
			BufferItems: 64,      // number of keys per Get buffer.
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate new cache for file store")
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

func (s *fileStore) buildEncryptor() error {
	exist, key, err := consulutil.GetStringValue(path.Join(consulutil.StoresPrefix, s.id, "key"))
	if err != nil {
		return errors.Wrapf(err, "failed to get withEncryption key for store with ID:%q", s.id)
	}
	s.encryptor, err = encryption.NewEncryptor(key)
	if err != nil {
		return errors.Wrapf(err, "failed to instantiate encryptor for store with ID:%q", s.id)
	}
	// save the encryption key if new
	if !exist {
		lock, _, err := s.getConsulLock()
		if err != nil {
			return err
		}
		defer lock.Unlock()
		err = consulutil.StoreConsulKeyAsString(path.Join(consulutil.StoresPrefix, s.id, "key"), s.encryptor.Key)
		if err != nil {
			return errors.Wrapf(err, "failed to save encryption key for store with ID:%q", s.id)
		}
	}
	return nil
}

func (s *fileStore) getConsulLock() (*api.Lock, <-chan struct{}, error) {
	lock, err := s.cc.LockOpts(&api.LockOptions{
		Key:          path.Join(consulutil.StoresPrefix, s.id, ".lock"),
		LockTryOnce:  true,
		LockWaitTime: 30 * time.Second,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get lock options for file store with ID:%q", s.id)
	}

	var lockCh <-chan struct{}
	for lockCh == nil {
		log.Debug("Try to acquire Consul lock for file store with ID:%q")
		lockCh, err = lock.Lock(nil)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed trying acquiring Consul lock for file store with ID:%q", s.id)
		}

	}
	log.Debug("Consul Lock for file store %q acquired", s.id)
	return lock, lockCh, nil
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

func (s *fileStore) Set(ctx context.Context, k string, v interface{}) error {
	if err := utils.CheckKeyAndValue(k, v); err != nil {
		return err
	}

	data, err := s.codec.Marshal(v)
	if err != nil {
		return err
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

func (s *fileStore) SetCollection(ctx context.Context, keyValues []*types.KeyValue) error {
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
				log.Debugf("Value has been retrieved from cache for key:%q", k)
				return true, s.codec.Unmarshal(data, v)
			}
			log.Printf("[WARNING] Failed to cast retrieved value from cache to bytes array for key:%q. Data will be retrieved from store.", k)
		}
	}

	filePath := s.buildFilePath(k, true)

	// Prepare file lock.
	lock := s.prepareFileLock(filePath)

	// File lock and file handling.
	lock.RLock()
	// Deferring the unlocking would lead to the unmarshalling being done during the lock, which is bad for performance.
	data, err := ioutil.ReadFile(filePath)
	lock.RUnlock()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// decrypt if necessary
	if s.withEncryption {
		data, err = s.encryptor.Decrypt(data)
		if err != nil {
			return false, err
		}
	}

	return true, s.codec.Unmarshal(data, v)
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
		// return the whole key path without store specific extension
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

func (s *fileStore) GetLastIndex(k string) (uint64, error) {
	// key can be directory or file, ie with or without extension
	// let's try first without extension as it can be the most current case
	fp := s.buildFilePath(k, false)
	fInfo, err := os.Stat(fp)

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
			} else {
				// File not found : the key doesn't exist
				return 0, nil
			}
		}
	}

	return uint64(fInfo.ModTime().UnixNano()), nil
}