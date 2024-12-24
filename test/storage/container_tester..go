package storage_test

import (
	"context"
	"fmt"
	"testing"
)

type ContainerTester interface {
	ClearDB(ctx context.Context) error
}

func Clear(ctx context.Context, t *testing.T, dlt ContainerTester) {
	err := dlt.ClearDB(ctx)

	if err != nil {
		t.Fatal(fmt.Errorf("failed to clear the database - %w", err))
	}
}
