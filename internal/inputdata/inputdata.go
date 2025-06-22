// Package inputdata - пакет для получения входных данных от клиента
package inputdata

import (
	"flag"
)

type InputData struct {
	ErrCode string
	Period string
	Discret string
	RATE100 float64
}

func New() *InputData {
	// код искомых ошибок
	var errCode string
	flag.StringVar(&errCode, "code", "403", "error code")

	// период, за который хотим посмотреть ошибки
	var period string
	flag.StringVar(&period, "p", "00 h", "time period")

	// дискретизация - какие временные промежутки хотим увидеть на гистограмме
	var discret string
	flag.StringVar(&discret, "d", "m", "sampling")
	
	return &InputData{
		ErrCode: errCode,
		Period: period,
		Discret: discret,
		RATE100: 100,
	}
}
