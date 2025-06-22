// Программа, которая рисует гистограмму с процентом ошибок за указанный временной период
package main

import (	
	"fmt"
	"log"
	"path/filepath"
	"strings"
	
	"github.com/Kostushka/logs/internal/histogram"
	"github.com/Kostushka/logs/internal/data"
	"github.com/Kostushka/logs/internal/inputdata"
)

// временно захардкордим путь до файла с логами
var basePath = "/home/kostushka/nlogs/logs/lgfile/"
var path = "vh442-20250425"

func main() {
	// сформировать структуру с входными данными
	inputData := inputdata.New()
	
	// формируем путь до файла
	filePath := filepath.Join(basePath, filepath.Clean(path))
	if !strings.HasPrefix(filePath, basePath) {
		log.Fatalf("invalid file path")
	}
	// получить структуру с данными о запросах и ошибках
	counter, err := data.GiveCountReqErr(inputData, filePath)
	if err != nil {
		log.Fatal(err)
	}
	// проверяем, что ошибки за выбранный период были
	if counter.MaxRate == 0 {
		fmt.Printf("ошибок с кодом ответа %q за период %s не обнаружено\n", inputData.ErrCode, inputData.Period)

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
	h.PrintHistogram(counter, data, inputData.Period)
}


