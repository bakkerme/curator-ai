package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dgraph-io/badger/v4"
)

type BadgerDB struct {
	db *badger.DB
}

func NewBadgerDB(path string) (*BadgerDB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable badger's default logger

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	return &BadgerDB{db: db}, nil
}

func (b *BadgerDB) Close() error {
	return b.db.Close()
}

func (b *BadgerDB) Set(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (b *BadgerDB) Get(key string, value interface{}) error {
	return b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, value)
		})
	})
}

func (b *BadgerDB) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (b *BadgerDB) List(prefix string, limit int) ([]string, error) {
	var keys []string
	
	return keys, b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		count := 0
		
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes) && (limit == 0 || count < limit); it.Next() {
			keys = append(keys, string(it.Item().Key()))
			count++
		}
		
		return nil
	})
}