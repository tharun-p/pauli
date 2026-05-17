package clickhouse

import (
	"testing"

	"github.com/tharun/pauli/internal/storage"
)

func TestRepositoryImplementsInterface(t *testing.T) {
	var _ storage.Repository = (*Repository)(nil)
}
