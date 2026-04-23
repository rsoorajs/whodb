package source

import (
	"context"
	"testing"
)

type lifecycleTestConnector struct {
	invalidated bool
	shutdown    bool
}

func (c *lifecycleTestConnector) Open(_ context.Context, _ TypeSpec, _ *Credentials) (SourceSession, error) {
	return nil, nil
}

func (c *lifecycleTestConnector) Invalidate(_ context.Context, _ TypeSpec, _ *Credentials) error {
	c.invalidated = true
	return nil
}

func (c *lifecycleTestConnector) Shutdown(_ context.Context) error {
	c.shutdown = true
	return nil
}

func TestInvalidateDelegatesToDriver(t *testing.T) {
	driverID := "test-lifecycle-invalidate"
	connector := &lifecycleTestConnector{}
	RegisterDriver(driverID, connector)

	err := Invalidate(context.Background(), TypeSpec{DriverID: driverID}, &Credentials{SourceType: "Test"})
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}
	if !connector.invalidated {
		t.Fatal("expected driver invalidation to be called")
	}
}

func TestShutdownDelegatesToDrivers(t *testing.T) {
	driverID := "test-lifecycle-shutdown"
	connector := &lifecycleTestConnector{}
	RegisterDriver(driverID, connector)

	err := Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if !connector.shutdown {
		t.Fatal("expected driver shutdown to be called")
	}
}
