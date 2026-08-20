package register

import (
	"sync"
	"testing"
)

func TestMigrationSourceRegistryReturnsDefensiveCopy(t *testing.T) {
	source := &SQLFS{}
	baseline := len(GetMigrationSources())
	AddMigrationSource(nil)
	if got := len(GetMigrationSources()); got != baseline {
		t.Fatalf("nil source changed registry length: got %d want %d", got, baseline)
	}
	AddMigrationSource(source)
	sources := GetMigrationSources()
	if len(sources) != baseline+1 || sources[len(sources)-1] == source {
		t.Fatalf("registered source missing: %#v", sources)
	}
	sources[len(sources)-1] = nil
	if got := GetMigrationSources()[baseline]; got == nil || got == source {
		t.Fatal("caller mutation must not change migration registry")
	}
}

func TestMigrationSourceRegistrySupportsConcurrentRegistrationAndReads(t *testing.T) {
	const writers = 16
	const readers = 16
	const readsPerReader = 100
	baseline := len(GetMigrationSources())
	var wg sync.WaitGroup
	for index := 0; index < writers; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			AddMigrationSource(&SQLFS{})
		}()
	}
	for index := 0; index < readers; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for read := 0; read < readsPerReader; read++ {
				for _, source := range GetMigrationSources() {
					if source == nil {
						t.Errorf("registry returned a nil migration source")
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	if got := len(GetMigrationSources()); got != baseline+writers {
		t.Fatalf("concurrent registry length=%d want=%d", got, baseline+writers)
	}
}

func TestMigrationSourceRegistryNeverInvokesModuleFactory(t *testing.T) {
	invoked := false
	AddModule(func(ctx interface{}) Module {
		invoked = true
		return Module{Name: "must-not-run-for-migration-registry"}
	})
	_ = GetMigrationSources()
	if invoked {
		t.Fatal("migration source registry must not instantiate a runtime module")
	}
}
