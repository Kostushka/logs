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

	"github.com/klauspost/compress/zstd"
)

// RATE100 константа для вычисления значения процента
const RATE100 = 100

// временно захардкордим путь до файла с логами
var basePath = "/home/kostushka/nlogs/logs/lgfile/"
var path = "vh442-20250425"

// временно захардкордим код искомых ошибок
var errCode = "500"

// временно захардкордим период
var period = "00 h"

// временно захардкордим дискретизацию
var discret = "m"
var discretNum = 60

// временно захардкордим ширину окна
var screenW = 200

// временно захардкордим высоту окна
var screenH = 50

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

	// расчеты для гистограммы
	data := calcHistogram(counter)

	// отрисовка гистограммы
	printHistogram(counter, data)
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
		if minute, err := strconv.Atoi(hm[1]); err == nil {
			// считаем кол-во запросов за конкретную минуту (от 0 по 59)
			c.req[minute]++
			// подсчитываем кол-во искомых ошибок
			calcCountErr(c.err, str, minute)
		}
	} else if c.req[0] != 0 {
		// кол-во запросов за выбранный период должно быть записано
		return false
	}

	return true
}

type dataHistogram struct {
	scale  bool
	width  int
	height int
}

func calcHistogram(c *countReqErr) *dataHistogram {
	// если макс. значение процента ошибок < 1, умножаем все значения на 100
	var scale bool
	// длина отрезка между двумя засечками
	var width int

	// максимальный процент ошибок не должен быть меньше 1
	if int(c.maxRate) < 1 {
		scale = true
		width = screenW / int(c.maxRate*RATE100)
	} else {
		width = screenW / int(c.maxRate)
	}

	// разбивка дискретизации с учетом высоты окна
	var height = discretNum
	for screenH/height < 1 {
		height /= 2
	}

	return &dataHistogram{
		scale:  scale,
		width:  width,
		height: height,
	}
}

// рисуем ось Y
func printY(c *countReqErr, data *dataHistogram) {
	var step int

	var p int

	for i, v := range c.rate {
		// шаг дискретизации
		if step == discretNum/data.height-1 {
			step = 0

			continue
		}

		step++

		// рисуем ось Y
		fmt.Printf("%3d%s├", i, discret)
		// процент ошибок не должен быть меньше 1
		if data.scale {
			p = int(v * RATE100)
		} else {
			p = int(v)
		}
		// отображаем процент ошибок за указанный период дискретизации
		for p > 0 {
			w := data.width
			for w > 0 {
				fmt.Printf("─")

				w--
			}

			p--
		}

		if data.scale {
			fmt.Printf(" %d%%\n", int(v*RATE100))
		} else {
			fmt.Printf(" %d%%\n", int(v))
		}
	}
}

const digitToNum = 9

// рисуем ось X
func printX(data *dataHistogram, maxRate int) {
	maxR := maxRate
	// рисуем ось X
	for i := 0; i < maxR; i++ {
		w := data.width
		w--

		for w > 0 {
			fmt.Printf("─")

			w--
		}

		fmt.Printf("┬")
	}

	if data.scale {
		// fmt.Printf(" %% (процент ошибок * 100)")
		fmt.Printf(" %%*100\n     ")
	} else {
		// fmt.Printf(" %% (процент ошибок)")
		fmt.Printf(" %%\n     ")
	}

	maxR = maxRate

	flag := false

	for i := 1; i <= maxR; i++ {
		if i > digitToNum {
			w := data.width
			if flag {
				// в отрезок между засечками попадает цифра, учитываем ее
				w -= 2
			} else {
				w--
			}
			// отрезок между засечками
			for w > 0 {
				fmt.Printf(" ")

				w--
			}
			// либо число, либо два пробела под число
			if i%5 == 0 {
				fmt.Printf("%d", i)

				flag = true
			} else {
				fmt.Printf("  ")
			}

			continue
		}

		w := data.width
		w--

		for w > 0 {
			fmt.Printf(" ")

			w--
		}

		fmt.Printf("%d", i)
	}
}

// рисуем гистограмму
func printHistogram(c *countReqErr, data *dataHistogram) {
	fmt.Printf("t (период): %s\n", period)
	// рисуем ось Y
	printY(c, data)

	fmt.Printf("    └")

	var maxRate int
	// максимальный процент ошибок не должен быть меньше 1
	if data.scale {
		maxRate = int(c.maxRate * RATE100)
	} else {
		maxRate = int(c.maxRate)
	}

	// рисуем ось X
	printX(data, maxRate)

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
		req:  make([]int, discretNum),
		err:  make([]int, discretNum),
		rate: make([]float64, discretNum),
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
		c.rate[i] = float64(c.err[i]) / float64(c.req[i]) * RATE100
		if c.maxRate < c.rate[i] {
			c.maxRate = c.rate[i]
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
