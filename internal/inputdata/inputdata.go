// Package inputdata - пакет для получения входных данных от клиента
package inputdata

import (
	"flag"
	"log"
	"strings"
	"time"
)

type InputData struct {
	ErrCode    string
	DataBefore time.Time
	DataAfter  time.Time
	Discret    string
	RATE100    float64
}

func New() *InputData {
	// код искомых ошибок
	var errCode string
	flag.StringVar(&errCode, "code", "403", "error code")

	// период, за который хотим посмотреть ошибки
	var period string
	flag.StringVar(&period, "p", "2025-05-25:04:01:00..2025-05-25:06:01:00", "time period")

	// разбиваем строку с периодом на две даты
	dates := strings.Split(period, "..")
	if len(dates) != 2 {
		log.Fatal("период должен быть задан согласно формату: yyyy-mm-dd:hh:mm:ss..yyyy-mm-dd:hh:mm:ss")
	}

	// формат времени
	const formTime = "2006-01-02:15:04:05"

	// парсим строки на даты в формате "дата"
	tbefor, err := time.Parse(formTime, dates[0])
	if err != nil {
		log.Fatal(err)
	}
	tafter, err := time.Parse(formTime, dates[1])
	if err != nil {
		log.Fatal(err)
	}

	// проверяем, что первая дата меньше второй
	if !tafter.After(tbefor) {
		log.Fatalf("первая дата периода %v должна быть меньше второй даты %v", tbefor, tafter)
	}

	// период не должен быть меньше 2 секунд
	dif := tafter.Sub(tbefor)
	if dif.Seconds() < 2 {
		log.Fatalf("период не должен быть меньше 2 секунд, период: %v секунд(а)", dif.Seconds())
	}

	// кол-во перепутанных по времени строк, которое может быть в логе и которое надо постараться учесть
	var mixlines int
	flag.IntVar(&mixlines, "mixlines", 500, "number of mixed up lines in the log")

	// дискретизация - какие временные промежутки хотим увидеть на гистограмме
	var discret string
	flag.StringVar(&discret, "d", "m", "sampling")

	flag.Parse()

	return &InputData{
		ErrCode:    errCode,
		DataBefore: tbefor,
		DataAfter:  tafter,
		Discret:    discret,
		RATE100:    100,
	}
}
