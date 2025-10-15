package contracts

// Optimizer defines the interface for optimizing the database.
type Optimizer interface {
	// Optimize performs the database optimization.
	Optimize() error
}
