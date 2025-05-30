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

	if counter.maxRate == 0 {
		fmt.Printf("ошибок с кодом ответа %q за период %s не обнаружено\n", errCode, period)
		return
	}
	// отрисовка гистограммы
	printHistogram(counter)
}

// структура с данными о кол-ве запросов, ошибок, % ошибок за выбранный период
type countReqErr struct {
	req     []int
	err     []int
	rate    []float64
	maxRate float64
}

func splitFile(c *countReqErr, str string) bool {
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
		if min, err := strconv.Atoi(hm[1]); err == nil {
			// считаем кол-во запросов за конкретную минуту (от 0 по 59)
			c.req[min]++
			// подсчитываем кол-во искомых ошибок
			calcCountErr(c.err, str, min)
		}
		// кол-во запросов за выбранный период должно быть записано
	} else if c.req[0] != 0 {
		return false
	}
	return true
}

func printHistogram(c *countReqErr) {
	fmt.Printf("t (период): %s\n", period)
	// если макс. значение процента ошибок < 1, умножаем все значения на 100
	var scale bool
	if int(c.maxRate) <= 1 {
		scale = true
	}
	var p int
	for i, v := range c.rate {
		fmt.Printf("%3d%s├", i, discret)
		if scale {
			p = int(v * 100)
		} else {
			p = int(v)
		}
		for p > 0 {
			fmt.Printf("──")
			p--
		}
		fmt.Println("")
	}
	fmt.Printf("    └")

	var max int
	if scale {
		max = int(c.maxRate * 100)
	} else {
		max = int(c.maxRate)
	}

	for max > 0 {
		fmt.Printf("─┬")
		max--
	}
	if scale {
		// fmt.Printf(" %% (процент ошибок * 100)")
		fmt.Printf(" %%*100\n    ")
	} else {
		// fmt.Printf(" %% (процент ошибок)")
		fmt.Printf(" %%\n    ")
	}

	if scale {
		max = int(c.maxRate * 100)
	} else {
		max = int(c.maxRate)
	}

	for i := 0; i <= max; i++ {
		if i > 9 {
			if i%5 == 0 {
				fmt.Printf("%d", i)
			} else {
				fmt.Printf("  ")
			}
			continue
		}
		fmt.Printf("%d ", i)

		// if i % 5 == 0 {
		// fmt.Printf("%d ", i)
		// } else {
		// fmt.Printf("  ")
		// }
	}
	fmt.Println("")
	fmt.Println("Расшифровка:")
	fmt.Println("%*100 - значения процента ошибок умноженных на 100")
	fmt.Println("% - значения процента ошибок")
}

// получение кол-ва запросов и ошибок
func giveCountReqErr(path string) (*countReqErr, error) {
	// формируем путь до файла
	filePath := filepath.Join(basePath, filepath.Clean(path))
	if !strings.HasPrefix(filePath, basePath) {
		return nil, fmt.Errorf("invalid file path")
	}
	// открываем файл с логами
	fd, err := os.Open(filePath) //nolint

	if err != nil {
		// проверяем, не имеет ли файл расширение .zst
		if errors.Is(err, fs.ErrNotExist) {
			// открываем файл с логами
			fd, err = os.Open(filePath + ".zst") //nolint
			// файл с логами должен быть
			if err != nil {
				log.Fatal(err)
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
	// закрываем дескриптор открытого файла
	defer closeFile(fd)

	// считаем кол-во запросов и ошибок
	count := calcCountReqErr(fd)
	return count, nil
}

// закрытие дескриптора открытого файла
func closeFile(fd io.Closer) {
	err := fd.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// подсчет кол-ва запросов и ошибок
func calcCountReqErr(fd io.Reader) *countReqErr {
	// структура для чтения файла лога
	s := bufio.NewScanner(fd)

	// струтура с кол-вом запросов и ошибок
	c := countReqErr{
		req:  make([]int, 60),
		err:  make([]int, 60),
		rate: make([]float64, 60),
	}

	// разделяет файл лога на строки
	for s.Scan() {
		// записать строку в переменную
		str := s.Text()
		// подсчет кол-ва запросов и ошибок
		if !splitFile(&c, str) {
			break
		}
	}

	// обработка ошибок, отличных от EOF
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

	// вычисляем процент ошибок за каждую временную единицу дискретизации и максимальный процент ошибок за выбранный период
	for i := range c.req {
		c.rate[i] = float64(c.err[i]) / float64(c.req[i]) * 100
		if c.maxRate < c.rate[i] {
			c.maxRate = c.rate[i]
		}
	}
	fmt.Println(c.rate)
	// fmt.Println(c.rate, c.maxRate)
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
