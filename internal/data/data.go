// Package data - пакет для получения структуры с данными о запросах и ошибках
package data

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kostushka/logs/internal/inputdata"
	"github.com/Kostushka/logs/internal/types"
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

func CalcDiscretNum(data *inputdata.InputData) {
	switch data.Discret {
	case "h":
	// переводим период в часы
	// кол-во часов == кол-во элементов в срезах в types.CountReqErr
	case "m":
	// переводим период в минуты
	// кол-во минут == кол-во элементов в срезах в types.CountReqErr
	// Example:
	// период: 00h или 14-17h или 250812 надо перевести в минуты
	//	00h[60 elem] 14-17h[240 elem] 250812[1440 elem]
	case "s":
	// переводим период в секунды
	// кол-во секунд == кол-во элементов в срезах в types.CountReqErr
	default:
		// переводим период в секунды
		// считаем высоту окна
		// проверяем, что гистограмма может быть отрисована на данной ширине и высоте окна терминала
		// кол-во секунд / высота окна == кол-во элементов в срезах в types.CountReqErr
	}
}

// получение данных о кол-ве запросов, ошибок, % ошибок и др.
func GiveCountReqErr(data *inputdata.InputData, pathDir string) (*types.CountReqErr, error) {

	// получаем данные файлов с логами для последующего их чтения
	fdList, err := openLogFiles(pathDir)

	if err != nil {
		return nil, err
	}

	// закрываем дескрипторы открытых файлов
	for _, fd := range fdList.fd {
		defer closeFile(fd)
	}

	// получаем данные о кол-ве запросов, ошибок, % ошибок и др.
	dataLog, err := calcCountReqErr(fdList, data)
	if err != nil {
		return nil, err
	}

	return dataLog, nil
}

type logFileData struct {
	fd   []*os.File
	name []string
}

// получаем данные файлов с логами для последующего их чтения
func openLogFiles(pathDir string) (*logFileData, error) {
	// получаем список всех файлов по заданному пути к корневому каталогу с логами
	files, err := os.ReadDir(pathDir)
	if err != nil {
		return nil, err
	}

	// список файловых дескрипторов и имен файлов с логами для последующего их чтения
	fdList := logFileData{}

	for _, file := range files {
		// здесь должна быть логика выбора из всех файлов нужных, согласно заданному периоду
		// пока костыль
		if strings.HasPrefix(file.Name(), "vh442-20250424") {
			// открываем файл с логами
			fd, err := os.Open(filepath.Join(pathDir, file.Name())) //nolint:gosec

			// файл с логами должен быть
			if err != nil {
				return nil, err
			}

			// добавляем файловый дескриптор и имя файла
			fdList.fd = append(fdList.fd, fd)
			fdList.name = append(fdList.name, file.Name())
		}
	}
	// хотя бы один файл с логами должен быть
	if len(fdList.fd) == 0 {
		return nil, fmt.Errorf("нет файлов логов, соответствующих заданному временному периоду")
	}
	return &fdList, nil
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
func calcCountReqErr(fdList *logFileData, data *inputdata.InputData) (*types.CountReqErr, error) {
	// струтура с кол-вом запросов и ошибок
	c := types.CountReqErr{
		Req:  make([]int, discretNum),
		Err:  make([]int, discretNum),
		Rate: make([]float64, discretNum),
	}

	// считаем данные по каждому файлу лога
	for i := 0; i < len(fdList.fd); i++ {
		var fd io.Reader = fdList.fd[i]

		// проверяем, не имеет ли файл расширение .zst
		if strings.HasSuffix(fdList.name[i], ".zst") {
			// декодируем формат zstd
			d, err := zstd.NewReader(fdList.fd[i])
			if err != nil {
				return nil, err
			}
			fd = d
		}

		// структура для чтения файла лога
		s := bufio.NewScanner(fd)

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
	}

	// вычисляем процент ошибок за каждую временную единицу дискретизации и максимальный процент ошибок за выбранный период
	for i := range c.Req {
		c.Rate[i] = float64(c.Err[i]) / float64(c.Req[i]) * data.RATE100
		if c.MaxRate < c.Rate[i] {
			c.MaxRate = c.Rate[i]
		}
	}

	return &c, nil
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
