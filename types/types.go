// Package types - пакет со структурами
package types

// структура с данными о кол-ве запросов, ошибок, % ошибок за выбранный период
type CountReqErr struct {
	Num     int
	Req     []int
	Err     []int
	Rate    []float64
	MaxRate float64
}
