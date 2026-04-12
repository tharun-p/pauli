package steps

import "context"

// Step is the contract for each unit in a linear chain. The runner passes *Env so steps share iteration context.
type Step interface {
	Async() bool
	// Run executes on the runner goroutine. Sync steps perform all work and must return enqueue=false.
	// Async steps return enqueue=true to schedule RunAsync on a worker, or false to skip (e.g. nothing to do this iteration).
	Run(env *Env) (enqueue bool, err error)
	// RunAsync runs on a worker after enqueue; only called when Async() is true and Run returned enqueue=true.
	RunAsync(ctx context.Context, env *Env) error
}
