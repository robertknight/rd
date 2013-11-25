package main

import (
	"encoding/gob"
	"log"
	"os"
)

type recentDirStore struct {
	path string
}

func NewRecentDirStore(path string) recentDirStore {
	return recentDirStore{path: path}
}

func (store *recentDirStore) Save(history map[string]DirUsage) {
	file, err := os.Create(store.path)
	if err != nil {
		log.Printf("Failed to open file to store map: %v", err)
		return
	}
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(history)
	if err != nil {
		log.Printf("Failed to store map: %v", err)
	}
}

func (store *recentDirStore) Load() map[string]DirUsage {
	result := map[string]DirUsage{}
	file, err := os.Open(store.path)
	if err != nil {
		log.Printf("Unable to load map: %v", err)
		return result
	}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		log.Printf("Failed to load map: %v", err)
		return result
	}
	log.Printf("Loaded %d entries", len(result))
	return result
}
