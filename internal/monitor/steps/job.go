package steps

// Job is one async step handed to the worker pool: RunAsync runs with Env captured from the runner iteration.
type Job struct {
	Step Step
	Env  Env
}
