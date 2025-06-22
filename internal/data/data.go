// Package data - пакет для получения структуры с данными о запросах и ошибках
package data

import (
	"os"
	"io"
	"io/fs"
	"errors"
	"strings"
	"fmt"
	"log"
	"strconv"
	"regexp"
	"bufio"

	"github.com/Kostushka/logs/internal/types"
	"github.com/Kostushka/logs/internal/inputdata"
	"github.com/klauspost/compress/zstd"
)

// регулярка для извлечения кода ошибки
const timeReg = `(\d{0,4}\.\d{0,4}|-)`
const errReg = `\[\d+/\w+/\d+:\d+:\d+[^]]+\] "[^"]*" ` + timeReg + ` ` + timeReg + ` (\d+)`

// создаем структуру для работы с регуляркой
var reErr = regexp.MustCompile(errReg)

// регулярка для получения временного периода
const pReg = `^\w* \d* (\d{2}:\d{2})`

// создаем структуру для работы с регуляркой
var reP = regexp.MustCompile(pReg)

// временно захардкордим дискретизацию
var discretNum = 60

// получение кол-ва запросов и ошибок
func GiveCountReqErr(data *inputdata.InputData, path string) (*types.CountReqErr, error) {
	// открываем файл с логами
	fd, err := os.Open(path) //nolint

	if err == nil {
		// закрываем дескриптор открытого файла
		defer closeFile(fd)

		// считаем кол-во запросов и ошибок
		count := calcCountReqErr(fd, data)

		return count, nil
	}

	// проверяем, не имеет ли файл расширение .zst
	if errors.Is(err, fs.ErrNotExist) {
		// открываем файл с логами
		fd, err = os.Open(path + ".zst") //nolint
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
		count := calcCountReqErr(d, data)

		return count, nil
	}

	return nil, err
}

func splitStr(c *types.CountReqErr, data *inputdata.InputData, str string) bool {
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
			calcCountErr(data, c.Err, str, minute)
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
func calcCountReqErr(fd io.Reader, data *inputdata.InputData) *types.CountReqErr {
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
		if !splitStr(&c, data, str) {
			break
		}
	}

	// обработка ошибок, отличных от EOF
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

	// вычисляем процент ошибок за каждую временную единицу дискретизации и максимальный процент ошибок за выбранный период
	for i := range c.Req {
		c.Rate[i] = float64(c.Err[i]) / float64(c.Req[i]) * data.RATE100
		if c.MaxRate < c.Rate[i] {
			c.MaxRate = c.Rate[i]
		}
	}

	return &c
}

// подсчет кол-ва искомых ошибок
func calcCountErr(data *inputdata.InputData, err []int, str string, i int) {
	// получаем срез выражений в () в совпадающих с регуляркой строках
	codeStat := reErr.FindStringSubmatch(str)
	// срез не должен быть пустым
	if len(codeStat) != 0 {
		// проверяем совпадения кода ответа с искомой ошибкой
		if codeStat[3] == data.ErrCode {
			// инкреметируем счетчик ошибок
			err[i]++
		}
	}
}


