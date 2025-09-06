// Программа, которая рисует гистограмму с процентом ошибок за указанный временной период
package main

import (
	"fmt"
	"log"

	"github.com/Kostushka/logs/internal/data"
	"github.com/Kostushka/logs/internal/histogram"
	"github.com/Kostushka/logs/internal/inputdata"
)

// временно захардкордим путь к корневому каталогу с логами
var path = "/home/kostushka/nlogs/logs/lgfile/"

func main() {
	// сформировать структуру с входными данными
	inputData := inputdata.New()

	// получить структуру с данными о запросах и ошибках
	counter, err := data.GiveCountReqErr(inputData, path)
	if err != nil {
		log.Fatal(err)
	}
	// проверяем, что ошибки за выбранный период были
	if counter.MaxRate == 0 {
		fmt.Printf("ошибок с кодом ответа %q не обнаружено\n", inputData.ErrCode)

		return
	}
	// создаем объект с данными для отрисовки гистограммы
	h, err := histogram.NewHistogram(inputData.Discret)
	if err != nil {
		log.Fatal(err)
	}

	// расчеты для гистограммы
	data, err := h.CalcHistogram(counter)
	if err != nil {
		log.Fatal(err)
	}

	// отрисовка гистограммы
	h.PrintHistogram(counter, data, inputData.DataBefore, inputData.DataAfter)
}
