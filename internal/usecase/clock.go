package usecase

import "time"

// Clock передаёт прикладным сценариям текущее время.
type Clock interface {
	Now() time.Time
}

// SystemClock получает время из системных часов.
type SystemClock struct{}

// Now возвращает текущее время в UTC.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
