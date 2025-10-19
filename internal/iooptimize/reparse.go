package iooptimize

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gnames/gndb/pkg/config"
)

// reparsed holds the data for a name_string being reparsed.
// This structure mirrors gnidump's reparsed struct for compatibility.
type reparsed struct {
	nameStringID                                  string
	name                                          string
	canonicalID, canonicalFullID, canonicalStemID sql.NullString
	canonical, canonicalFull, canonicalStem       string
	bacteria, surrogate, virus                    bool
	parseQuality                                  int
}

// loadNamesForReparse loads all name_strings from database for reparsing.
// It queries the database and sends each name_string to the input channel.
// Progress is logged every 100,000 names.
//
// Reference: gnidump loadReparse() in db_reparse.go
func loadNamesForReparse(
	ctx context.Context,
	optimizer *OptimizerImpl,
	chIn chan<- reparsed,
) error {
	pool := optimizer.operator.Pool()
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	q := `
SELECT
	id, name, canonical_id, canonical_full_id, canonical_stem_id, bacteria,
	virus, surrogate, parse_quality
FROM name_strings
`
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return NewReparseQueryError(err)
	}
	defer rows.Close()

	var count int
	timeStart := time.Now().UnixNano()

	for rows.Next() {
		count++
		res := reparsed{}
		err = rows.Scan(
			&res.nameStringID, &res.name, &res.canonicalID,
			&res.canonicalFullID, &res.canonicalStemID,
			&res.bacteria, &res.virus, &res.surrogate,
			&res.parseQuality,
		)
		if err != nil {
			return NewReparseScanError(err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chIn <- res
		}

		// Progress tracking: log every 100,000 names
		if count%100_000 == 0 {
			timeSpent := float64(time.Now().UnixNano()-timeStart) / 1_000_000_000
			speed := int64(float64(count) / timeSpent)
			fmt.Fprintf(os.Stderr, "\r%s", strings.Repeat(" ", 40))
			fmt.Fprintf(os.Stderr, "\rLoaded %s names, %s names/sec",
				humanize.Comma(int64(count)), humanize.Comma(speed))
		}
	}

	// Check for errors from iteration
	if err := rows.Err(); err != nil {
		return NewReparseIterationError(err)
	}

	// Clear progress line
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 40))
	return nil
}

// reparseNames orchestrates the name reparsing workflow.
// This is the main function that will be called from Optimize().
// It will be fully implemented in T008 after all worker functions are ready.
func reparseNames(ctx context.Context, optimizer *OptimizerImpl, cfg *config.Config) error {
	// This function will be implemented in T008 after T004-T007 are complete
	// For now, it calls the placeholder to maintain TDD red phase
	return errNotImplemented("reparseNames")
}
