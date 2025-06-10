// Программа, которая рисует гистограмму с процентом ошибок за указанный временной период
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kostushka/logs/histogram"
	"github.com/Kostushka/logs/types"
	"github.com/klauspost/compress/zstd"
)

// временно захардкордим путь до файла с логами
var basePath = "/home/kostushka/nlogs/logs/lgfile/"
var path = "vh442-20250425"

// временно захардкордим код искомых ошибок
var errCode = "500"

// временно захардкордим период
var period = "00 h"

// временно захардкордим дискретизацию
var discret = "m"

// временно захардкордим дискретизацию
var discretNum = 60

// RATE100 константа для вычисления значения процента
const RATE100 = 100

// регулярка для извлечения кода ошибки
const timeReg = `(\d{0,4}\.\d{0,4}|-)`
const errReg = `\[\d+/\w+/\d+:\d+:\d+[^]]+\] "[^"]*" ` + timeReg + ` ` + timeReg + ` (\d+)`

// создаем структуру для работы с регуляркой
var reErr = regexp.MustCompile(errReg)

// регулярка для получения временного периода
const pReg = `^\w* \d* (\d{2}:\d{2})`

// создаем структуру для работы с регуляркой
var reP = regexp.MustCompile(pReg)

func main() {
	// получить структуру с данными о запросах и ошибках
	counter, err := giveCountReqErr(path)
	if err != nil {
		log.Fatal(err)
	}

	if counter.MaxRate == 0 {
		fmt.Printf("ошибок с кодом ответа %q за период %s не обнаружено\n", errCode, period)

		return
	}

	h := histogram.NewHistogram(discret)

	// расчеты для гистограммы
	data := h.CalcHistogram(counter)

	// отрисовка гистограммы
	h.PrintHistogram(counter, data)
}

// получение кол-ва запросов и ошибок
func giveCountReqErr(path string) (*types.CountReqErr, error) {
	// формируем путь до файла
	filePath := filepath.Join(basePath, filepath.Clean(path))
	if !strings.HasPrefix(filePath, basePath) {
		return nil, fmt.Errorf("invalid file path")
	}
	// открываем файл с логами
	fd, err := os.Open(filePath) //nolint

	if err == nil {
		// закрываем дескриптор открытого файла
		defer closeFile(fd)

		// считаем кол-во запросов и ошибок
		count := calcCountReqErr(fd)

		return count, nil
	}

	// проверяем, не имеет ли файл расширение .zst
	if errors.Is(err, fs.ErrNotExist) {
		// открываем файл с логами
		fd, err = os.Open(filePath + ".zst") //nolint
		// файл с логами должен быть
		if err != nil {
			return nil, err
		}
		// закрываем дескриптор открытого файла
		defer closeFile(fd)
		// декодируем формат zstd
		d, err := zstd.NewReader(fd)
		if err != nil {
			return nil, err
		}
		// считаем кол-во запросов и ошибок
		count := calcCountReqErr(d)

		return count, nil
	}

	return nil, err
}

func splitStr(c *types.CountReqErr, str string) bool {
	// извлекаем время из строки лога
	time := reP.FindStringSubmatch(str)
	// время в строке лога должно быть
	if len(time) == 0 {
		fmt.Println("некорректный формат лога")

		return true
	}

	// получаем массив с разделением на час и минуты
	hm := strings.Split(time[1], ":")
	// берем данные за конкретный час
	if hm[0] == "00" {
		// преобразуем строку с минутой в число
		if minute, err := strconv.Atoi(hm[1]); err == nil {
			// подсчитываем кол-во записей
			if c.Req[minute] == 0 {
				c.Num++
			}
			// считаем кол-во запросов за конкретную минуту (от 0 по 59)
			c.Req[minute]++
			// подсчитываем кол-во искомых ошибок
			calcCountErr(c.Err, str, minute)
		}
	} else if c.Req[0] != 0 {
		// кол-во запросов за выбранный период должно быть записано
		return false
	}

	return true
}

// закрытие дескриптора открытого файла
func closeFile(fd io.Closer) {
	err := fd.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// подсчет кол-ва запросов и ошибок
func calcCountReqErr(fd io.Reader) *types.CountReqErr {
	// структура для чтения файла лога
	s := bufio.NewScanner(fd)

	// струтура с кол-вом запросов и ошибок
	c := types.CountReqErr{
		Req:  make([]int, discretNum),
		Err:  make([]int, discretNum),
		Rate: make([]float64, discretNum),
	}

	// разделяет файл лога на строки
	for s.Scan() {
		// записать строку в переменную
		str := s.Text()
		// подсчет кол-ва запросов и ошибок
		if !splitStr(&c, str) {
			break
		}
	}

	// обработка ошибок, отличных от EOF
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

	// вычисляем процент ошибок за каждую временную единицу дискретизации и максимальный процент ошибок за выбранный период
	for i := range c.Req {
		c.Rate[i] = float64(c.Err[i]) / float64(c.Req[i]) * RATE100
		if c.MaxRate < c.Rate[i] {
			c.MaxRate = c.Rate[i]
		}
	}

	return &c
}

// подсчет кол-ва искомых ошибок
func calcCountErr(err []int, str string, i int) {
	// получаем срез выражений в () в совпадающих с регуляркой строках
	codeStat := reErr.FindStringSubmatch(str)
	// срез не должен быть пустым
	if len(codeStat) != 0 {
		// проверяем совпадения кода ответа с искомой ошибкой
		if codeStat[3] == errCode {
			// инкреметируем счетчик ошибок
			err[i]++
		}
	}
}
