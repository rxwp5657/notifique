package main

import "github.com/notifique/internal"

func getStorage() internal.InMemoryStorage {
	return internal.MakeInMemoryStorage()
}
