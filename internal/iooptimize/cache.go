package iooptimize

import (
	"log/slog"

	"github.com/dgraph-io/badger/v4"
	"github.com/gnames/gnfmt"
	"github.com/gnames/gnparser/ent/parsed"
	"github.com/gnames/gnsys"
)

// parsedData holds the minimal parsed information we need to cache.
// This matches the structure used in gnidump for efficient storage.
type parsedData struct {
	ID              string
	CanonicalSimple string
	CanonicalFull   string
}

// CacheManager manages an ephemeral Badger v4 key-value store for caching
// parsed name results during optimization. The cache is stored at
// ~/.cache/gndb/optimize/ and should be cleaned up after optimization completes.
type CacheManager struct {
	dir string
	db  *badger.DB
}

// NewCacheManager creates a new cache manager at the specified directory.
// It creates the directory if it doesn't exist and cleans any existing data.
func NewCacheManager(cacheDir string) (*CacheManager, error) {
	cm := &CacheManager{
		dir: cacheDir,
	}

	err := gnsys.MakeDir(cacheDir)
	if err != nil {
		slog.Error("Cannot create cache directory", "error", err, "dir", cacheDir)
		return nil, err
	}

	err = gnsys.CleanDir(cacheDir)
	if err != nil {
		slog.Error("Cannot clean cache directory", "error", err, "dir", cacheDir)
		return nil, err
	}

	return cm, nil
}

// Open opens the Badger database for the cache.
func (c *CacheManager) Open() error {
	if c.db != nil {
		slog.Warn("Cache database is already open")
		return nil
	}

	options := badger.DefaultOptions(c.dir)
	options.Logger = nil // Disable badger's internal logging

	db, err := badger.Open(options)
	if err != nil {
		slog.Error("Cannot open cache database", "error", err, "dir", c.dir)
		return err
	}

	c.db = db
	slog.Info("Cache database opened", "dir", c.dir)
	return nil
}

// Close closes the Badger database.
func (c *CacheManager) Close() error {
	if c.db == nil {
		slog.Warn("Cache database is already closed")
		return nil
	}

	err := c.db.Close()
	c.db = nil

	if err != nil {
		slog.Error("Cannot close cache database", "error", err)
		return err
	}

	slog.Info("Cache database closed")
	return nil
}

// StoreParsed stores a parsed name result in the cache, encoded with GOB.
// The key is the name_string_id (UUID as string).
func (c *CacheManager) StoreParsed(nameStringID string, parsed *parsed.Parsed) error {
	if c.db == nil {
		return NewCacheNotOpenError()
	}

	// Extract minimal data for caching
	var canonicalSimple, canonicalFull string
	if parsed.Parsed {
		canonicalSimple = parsed.Canonical.Simple
		canonicalFull = parsed.Canonical.Full
	}

	data := parsedData{
		ID:              parsed.VerbatimID,
		CanonicalSimple: canonicalSimple,
		CanonicalFull:   canonicalFull,
	}

	// Encode with GOB
	enc := gnfmt.GNgob{}
	valBytes, err := enc.Encode(data)
	if err != nil {
		slog.Error("Cannot encode parsed data", "error", err, "id", nameStringID)
		return err
	}

	// Store in Badger transaction
	err = c.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(nameStringID), valBytes)
	})

	if err != nil {
		slog.Error("Cannot store parsed data", "error", err, "id", nameStringID)
		return err
	}

	return nil
}

// GetParsed retrieves a parsed name result from the cache and decodes it from GOB.
// Returns nil if the key is not found.
func (c *CacheManager) GetParsed(nameStringID string) (*parsedData, error) {
	if c.db == nil {
		return nil, NewCacheNotOpenError()
	}

	var valBytes []byte

	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(nameStringID))
		if err == badger.ErrKeyNotFound {
			return nil // Not an error, just not found
		}
		if err != nil {
			return err
		}

		valBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		slog.Error("Cannot retrieve parsed data", "error", err, "id", nameStringID)
		return nil, err
	}

	if valBytes == nil {
		// Key not found
		return nil, nil
	}

	// Decode with GOB
	enc := gnfmt.GNgob{}
	var data parsedData
	err = enc.Decode(valBytes, &data)
	if err != nil {
		slog.Error("Cannot decode parsed data", "error", err, "id", nameStringID)
		return nil, err
	}

	return &data, nil
}

// Cleanup closes the database and removes the cache directory.
// This should be called when optimization is complete.
func (c *CacheManager) Cleanup() error {
	// Close database first
	if err := c.Close(); err != nil {
		return err
	}

	// Remove directory
	err := gnsys.CleanDir(c.dir)
	if err != nil {
		slog.Error("Cannot remove cache directory", "error", err, "dir", c.dir)
		return err
	}

	slog.Info("Cache cleaned up", "dir", c.dir)
	return nil
}
