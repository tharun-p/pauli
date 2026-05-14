package realtime

func validatorIndexWatched(validators []uint64, index uint64) bool {
	for _, v := range validators {
		if v == index {
			return true
		}
	}
	return false
}
