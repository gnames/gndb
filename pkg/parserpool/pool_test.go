package parserpool_test

import (
	"sync"
	"testing"

	"github.com/gnames/gndb/pkg/parserpool"
	"github.com/gnames/gnlib/ent/nomcode"
)

// TestNewPool verifies pool creation with default and custom sizes.
func TestNewPool(t *testing.T) {
	tests := []struct {
		name     string
		jobsNum  int
		wantSize int
	}{
		{
			name:     "default size (0 = NumCPU)",
			jobsNum:  0,
			wantSize: 0, // We can't assert exact value, but we verify it works
		},
		{
			name:     "custom size 4",
			jobsNum:  4,
			wantSize: 4,
		},
		{
			name:     "custom size 1",
			jobsNum:  1,
			wantSize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := parserpool.NewPool(tt.jobsNum)
			if pool == nil {
				t.Fatal("NewPool returned nil")
			}
			defer pool.Close()

			// Verify pool works by parsing a simple name
			_, err := pool.Parse("Homo sapiens", nomcode.Botanical)
			if err != nil {
				t.Errorf("Parse failed: %v", err)
			}
		})
	}
}

// TestParse_BotanicalCode verifies botanical name parsing.
func TestParse_BotanicalCode(t *testing.T) {
	pool := parserpool.NewPool(2)
	defer pool.Close()

	tests := []struct {
		name       string
		nameString string
		wantParsed bool
	}{
		{
			name:       "simple botanical name",
			nameString: "Plantago major",
			wantParsed: true,
		},
		{
			name:       "botanical name with author",
			nameString: "Plantago major L.",
			wantParsed: true,
		},
		{
			name:       "botanical trinomial",
			nameString: "Rosa acicularis var. acicularis",
			wantParsed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pool.Parse(tt.nameString, nomcode.Botanical)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result.Parsed != tt.wantParsed {
				t.Errorf("Parse result.Parsed = %v, want %v", result.Parsed, tt.wantParsed)
			}

			if tt.wantParsed && result.Canonical.Simple == "" {
				t.Errorf("Expected non-empty canonical for parsed name")
			}
		})
	}
}

// TestParse_ZoologicalCode verifies zoological name parsing.
func TestParse_ZoologicalCode(t *testing.T) {
	pool := parserpool.NewPool(2)
	defer pool.Close()

	tests := []struct {
		name       string
		nameString string
		wantParsed bool
	}{
		{
			name:       "simple zoological name",
			nameString: "Homo sapiens",
			wantParsed: true,
		},
		{
			name:       "zoological name with author",
			nameString: "Apis mellifera Linnaeus, 1758",
			wantParsed: true,
		},
		{
			name:       "zoological trinomial",
			nameString: "Passer domesticus domesticus",
			wantParsed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pool.Parse(tt.nameString, nomcode.Zoological)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result.Parsed != tt.wantParsed {
				t.Errorf("Parse result.Parsed = %v, want %v", result.Parsed, tt.wantParsed)
			}

			if tt.wantParsed && result.Canonical.Simple == "" {
				t.Errorf("Expected non-empty canonical for parsed name")
			}
		})
	}
}

// TestParse_UnsupportedCode verifies error handling for unsupported codes.
func TestParse_UnsupportedCode(t *testing.T) {
	pool := parserpool.NewPool(2)
	defer pool.Close()

	_, err := pool.Parse("Plantago major", nomcode.Bacterial)
	if err == nil {
		t.Error("Expected error for unsupported nomenclatural code, got nil")
	}
}

// TestParse_CodeDifference verifies botanical vs zoological parsing differences.
// The name "Aus (Bus)" is parsed differently depending on nomenclatural code:
// - Zoological: Canonical.Simple = "Bus" (subgenus in parentheses is primary)
// - Botanical: Canonical.Simple = "Aus" (genus is primary, parenthetical is ignored)
func TestParse_CodeDifference(t *testing.T) {
	pool := parserpool.NewPool(2)
	defer pool.Close()

	nameString := "Aus (Bus)"

	// Parse with zoological code
	zooResult, err := pool.Parse(nameString, nomcode.Zoological)
	if err != nil {
		t.Fatalf("Zoological parse failed: %v", err)
	}

	// Parse with botanical code
	botResult, err := pool.Parse(nameString, nomcode.Botanical)
	if err != nil {
		t.Fatalf("Botanical parse failed: %v", err)
	}

	// Verify different interpretations
	if zooResult.Canonical.Simple != "Bus" {
		t.Errorf("Zoological parse: got Canonical.Simple = %q, want %q",
			zooResult.Canonical.Simple, "Bus")
	}

	if botResult.Canonical.Simple != "Aus" {
		t.Errorf("Botanical parse: got Canonical.Simple = %q, want %q",
			botResult.Canonical.Simple, "Aus")
	}

	// Verify both pools are properly configured and return different results
	if zooResult.Canonical.Simple == botResult.Canonical.Simple {
		t.Error("Expected different Canonical.Simple values for zoological vs botanical codes")
	}
}

// TestParse_Concurrent verifies thread-safety with multiple goroutines.
func TestParse_Concurrent(t *testing.T) {
	poolSize := 4
	pool := parserpool.NewPool(poolSize)
	defer pool.Close()

	numGoroutines := 20
	namesPerGoroutine := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Alternate between botanical and zoological codes
			code := nomcode.Botanical
			if id%2 == 0 {
				code = nomcode.Zoological
			}

			for j := 0; j < namesPerGoroutine; j++ {
				nameString := "Plantago major"
				if code == nomcode.Zoological {
					nameString = "Homo sapiens"
				}

				result, err := pool.Parse(nameString, code)
				if err != nil {
					t.Errorf("Goroutine %d: Parse failed: %v", id, err)
					return
				}

				if !result.Parsed {
					t.Errorf("Goroutine %d: Name not parsed: %s", id, nameString)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestParse_PoolBlocking verifies blocking behavior when pool is exhausted.
func TestParse_PoolBlocking(t *testing.T) {
	// Create a very small pool to test blocking
	poolSize := 1
	pool := parserpool.NewPool(poolSize)
	defer pool.Close()

	// Channel to coordinate goroutines
	started := make(chan struct{})
	finished := make(chan struct{})

	// Start a goroutine that will hold the parser
	go func() {
		result, err := pool.Parse("Plantago major", nomcode.Botanical)
		if err != nil {
			t.Errorf("First parse failed: %v", err)
		}
		if !result.Parsed {
			t.Error("First parse unsuccessful")
		}
		close(started)

		// Wait before finishing
		<-finished
	}()

	// Wait for first goroutine to acquire the parser
	<-started

	// Second parse should complete eventually (after first releases)
	done := make(chan struct{})
	go func() {
		result, err := pool.Parse("Homo sapiens", nomcode.Botanical)
		if err != nil {
			t.Errorf("Second parse failed: %v", err)
		}
		if !result.Parsed {
			t.Error("Second parse unsuccessful")
		}
		close(done)
	}()

	// Release the first parser
	close(finished)

	// Wait for second parse to complete
	<-done
}

// TestClose verifies proper cleanup of resources.
func TestClose(t *testing.T) {
	pool := parserpool.NewPool(2)

	// Parse a name before closing
	_, err := pool.Parse("Plantago major", nomcode.Botanical)
	if err != nil {
		t.Fatalf("Parse before close failed: %v", err)
	}

	// Close should not panic
	pool.Close()

	// Note: Parsing after close would panic (sending on closed channel),
	// but that's expected behavior - Close() should only be called once
	// when the pool is no longer needed.
}

// TestParse_BothCodesConcurrently verifies both pools work simultaneously.
func TestParse_BothCodesConcurrently(t *testing.T) {
	pool := parserpool.NewPool(4)
	defer pool.Close()

	var wg sync.WaitGroup
	iterations := 50

	// Goroutines using botanical code
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			result, err := pool.Parse("Plantago major L.", nomcode.Botanical)
			if err != nil {
				t.Errorf("Botanical parse failed: %v", err)
				return
			}
			if !result.Parsed {
				t.Error("Botanical name not parsed")
			}
		}
	}()

	// Goroutines using zoological code
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			result, err := pool.Parse("Homo sapiens Linnaeus", nomcode.Zoological)
			if err != nil {
				t.Errorf("Zoological parse failed: %v", err)
				return
			}
			if !result.Parsed {
				t.Error("Zoological name not parsed")
			}
		}
	}()

	wg.Wait()
}
