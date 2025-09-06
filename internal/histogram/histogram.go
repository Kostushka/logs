// Package histogram - пакет для отрисовки гистограммы
package histogram

import (
	"fmt"
	"time"

	"golang.org/x/term"

	"github.com/Kostushka/logs/internal/types"
)

type histogram struct {
	rate100     float64
	screenW     int
	screenH     int
	discretNum  string
	discretName string
}

// функция-конструктор для создания объекта с данными для отрисовки гистограммы
func NewHistogram(dName string) (*histogram, error) {
	// высчитываем ширину и высоту экрана терминала
	width, height, err := term.GetSize(0)
	if err != nil {
		return nil, err
	}

	return &histogram{
		rate100: 100,
		screenW: width,
		screenH: height,
		// временно захардкордим дискретизацию
		discretNum: "m",
		// дискретизация
		discretName: dName,
	}, nil
}

// данные для отрисовки гистограммы
type dataHistogram struct {
	scale   bool
	width   int
	height  int
	maxRate float64
}

// вычисление данных для отрисовки гистограммы
func (h *histogram) CalcHistogram(c *types.CountReqErr) (*dataHistogram, error) {
	// если макс. значение процента ошибок < 1, умножаем все значения на 100
	var scale bool
	// длина отрезка между двумя засечками
	var width float64

	// шаг дискретизации с учетом высоты окна (по умолчанию)
	// 10 строк из высоты окна пойдет на служебную инфо
	height := h.screenH - 10
	// здесь должна быть логика, где мы учитываем указанный период и переводим его в секунды (пока захардкордим час в секундах)
	period := 3600
	// рассчитываем, сколько секунд приходится на каждую строку с учетом высоты окна
	sec := period / height
	// определяем шаг дискретизации (в одной строке отображаются данные за discretNum минут)
	discretNum := sec / 60

	// получаем максимальный процент ошибок с учетом определенного шага дискретизации
	maxRate := calcMaxRate(c, discretNum)

	// максимальный процент ошибок не должен быть меньше 1
	var maxR float64
	if int(maxRate) < 1 {
		scale = true
		maxR = maxRate * h.rate100
	} else {
		maxR = maxRate
	}
	// вычисляем ширину между засечками
	width = float64(h.screenW) / maxR

	// учитываем доп. отображаемые данные
	for h.screenW-int(maxR)*int(width) < 15 {
		width -= 1
	}

	if width < 1 {
		return nil, fmt.Errorf("Ширина окна терминала слишком мала для отрисовки гистограммы\n")
	}

	return &dataHistogram{
		scale:   scale,
		width:   int(width),
		height:  discretNum,
		maxRate: maxRate,
	}, nil
}

// расчет максимального процента ошибок с учетом определенного шага дискретизации
func calcMaxRate(c *types.CountReqErr, height int) float64 {
	var step int
	var max, sum float64

	flag := false

	for i, v := range c.Rate {
		if i == c.Num {
			break
		}
		// шаг дискретизации
		if step != height && flag {
			step++
			// суммируем неотображенные на графике проценты ошибок
			sum += v

			continue
		}
		sum += v
		if max < sum {
			max = sum
		}
		sum = 0
		step = 0
		flag = true
		// учитываем также последние данные, которые попадают в шаг дискретизации
		if len(c.Rate)-1-i <= height {
			for j := i; j < len(c.Rate)-1; j++ {
				sum += c.Rate[j]
			}
			if max < sum {
				max = sum
			}
			return max
		}
	}

	return max
}

const (
	Yellow = "\033[33m"
	Reset  = "\033[0m"
)

// рисуем ось Y
func (h *histogram) printY(c *types.CountReqErr, data *dataHistogram) {
	var step int
	var sum float64

	flag := false

	for i, v := range c.Rate {
		// ориентируемся на кол-во имеющихся записей с кол-вом запросов
		if i == c.Num {
			break
		}

		// шаг дискретизации
		if step != data.height && flag {
			step++
			// суммируем неотображенные на графике проценты ошибок
			sum += v

			continue
		}

		step = 0

		// учитываем неотображенные на графике проценты ошибок
		sum += v

		// рисуем ось Y
		fmt.Printf("%2d%s┠", i, h.discretName)

		// рисуем график за период дискретизации
		displayY(data, sum, h.rate100)

		sum = 0
		flag = true

		// отображаем также последние данные, которые попадают в шаг дискретизации
		if len(c.Rate)-1-i <= data.height {
			j := i + 1
			for ; j < len(c.Rate); j++ {
				sum += c.Rate[j]
			}
			// рисуем ось Y
			fmt.Printf("%2d%s┠", j-1, h.discretName)

			// рисуем график за период дискретизации
			displayY(data, sum, h.rate100)

			sum = 0
		}
	}
}

func displayY(data *dataHistogram, sum, rate100 float64) {
	var p int

	// процент ошибок не должен быть меньше 1
	if data.scale {
		p = int(sum * rate100)
	} else {
		p = int(sum)
	}

	printP := p

	// отображаем процент ошибок за указанный период дискретизации
	for p > 0 {
		w := data.width
		for w > 0 {
			fmt.Printf(Yellow + "─" + Reset)

			w--
		}

		p--
	}

	fmt.Printf(" %d%%\n", printP)
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
			fmt.Printf("━")

			w--
		}

		fmt.Printf("┯")
	}

	if data.scale {
		// fmt.Printf(" %% (процент ошибок * 100)")
		fmt.Printf(" %%*100\n    ")
	} else {
		// fmt.Printf(" %% (процент ошибок)")
		fmt.Printf(" %%\n    ")
	}

	flag := false

	// минимальное расстояние между засечками
	if data.width < 2 {
		for i := 1; i <= maxR; i++ {
			w := data.width
			if flag == true && i > digitToNum {
				w--
				flag = false
			}
			if i%5 == 0 {
				fmt.Printf("%d", i)
				flag = true
			} else {
				// отрезок между засечками
				for w > 0 {
					fmt.Printf(" ")

					w--
				}
			}
		}
		return
	}

	maxR = maxRate

	flag = false

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
func (h *histogram) PrintHistogram(c *types.CountReqErr, data *dataHistogram, before, after time.Time) {
	fmt.Printf("t (период): [%v] [%v]\n", before, after)
	// рисуем ось Y
	h.printY(c, data)

	fmt.Printf("   ┗")

	var maxRate int
	// максимальный процент ошибок не должен быть меньше 1
	if data.scale {
		maxRate = int(data.maxRate * h.rate100)
	} else {
		maxRate = int(data.maxRate)
	}

	// рисуем ось X
	printX(data, maxRate)

	fmt.Println("")
	fmt.Println("Расшифровка:")
	fmt.Println("%*100 - значения процента ошибок умноженных на 100")
	fmt.Println("% - значения процента ошибок")
}
