package service

// HealthReady reports whether the database is ready.
func (s Service[R]) HealthReady() bool {
	if err := s.db.Ready(); err != nil {
		return false
	}
	return true
}
