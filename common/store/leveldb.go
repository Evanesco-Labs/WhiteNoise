package store

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDBStore struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

// used to compute the size of bloom filter bits array .
// too small will lead to high false positive rate.
const BITSPERKEY = 10

//NewLevelDBStore return LevelDBStore instance
func NewLevelDBStore(file string) (*LevelDBStore, error) {
	// default Options
	o := opt.Options{
		NoSync: false,
		Filter: filter.NewBloomFilter(BITSPERKEY),
	}
	db, err := leveldb.OpenFile(file, &o)
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	if err != nil {
		return nil, err
	}

	return &LevelDBStore{
		db: db,
	}, nil
}

//Put a key-value pair to leveldb
func (self *LevelDBStore) Put(key []byte, value []byte) error {
	return self.db.Put(key, value, nil)
}

//Get the value of a key from leveldb
func (self *LevelDBStore) Get(key []byte) ([]byte, error) {
	return self.db.Get(key, nil)
}

//Has return whether the key is exist in leveldb
func (self *LevelDBStore) Has(key []byte) (bool, error) {
	return self.db.Has(key, nil)
}

//Delete the the in leveldb
func (self *LevelDBStore) Delete(key []byte) error {
	return self.db.Delete(key, nil)
}

// QueryKeysByPrefix. find all keys by prefix
func (self *LevelDBStore) QueryKeysByPrefix(prefix []byte) ([][]byte, error) {
	iter := self.db.NewIterator(util.BytesPrefix(prefix), nil)
	keys := make([][]byte, 0)
	for iter.Next() {
		keys = append(keys, iter.Key())
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// QueryStringKeysByPrefix. find all keys by prefix
func (self *LevelDBStore) QueryStringKeysByPrefix(prefix []byte) ([]string, error) {
	iter := self.db.NewIterator(util.BytesPrefix(prefix), nil)
	keys := make([]string, 0)
	for iter.Next() {
		keys = append(keys, string(iter.Key()))
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

//Close close leveldb
func (self *LevelDBStore) Close() error {
	return self.db.Close()
}

//NewBatch start commit batch
func (self *LevelDBStore) NewBatch() {
	self.batch = new(leveldb.Batch)
}

//BatchPut put a key-value pair to leveldb batch
func (self *LevelDBStore) BatchPut(key []byte, value []byte) {
	self.batch.Put(key, value)
}

//BatchDelete delete a key to leveldb batch
func (self *LevelDBStore) BatchDelete(key []byte) {
	self.batch.Delete(key)
}

//BatchCommit commit batch to leveldb
func (self *LevelDBStore) BatchCommit() error {
	err := self.db.Write(self.batch, nil)
	if err != nil {
		return err
	}
	self.batch = nil
	return nil
}
