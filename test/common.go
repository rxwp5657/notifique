package main

import "github.com/notifique/internal/storage"

func getStorage() storage.InMemoryStorage {
	return storage.MakeInMemoryStorage()
}
