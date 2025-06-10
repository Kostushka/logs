// Package histogram - пакет для отрисовки гистограммы
package histogram

import (
	"fmt"

	"github.com/Kostushka/logs/types"
)

type histogram struct {
	rate100     float64
	screenW     int
	screenH     int
	discretNum  int
	discretName string
}

// функция-конструктор для создания объекта с данными для отрисовки гистограммы
func NewHistogram(dName string) *histogram {
	return &histogram{
		rate100: 100,
		// временно захардкордим ширину окна
		screenW: 200,
		// временно захардкордим высоту окна
		screenH: 50,
		// временно захардкордим дискретизацию
		discretNum: 60,
		// дискретизация
		discretName: dName,
	}
}

// временно захардкордим период
var period = "00 h"

// данные для отрисовки гистограммы
type dataHistogram struct {
	scale  bool
	width  int
	height int
}

// вычисление данных для отрисовки гистограммы
func (h *histogram) CalcHistogram(c *types.CountReqErr) *dataHistogram {
	// если макс. значение процента ошибок < 1, умножаем все значения на 100
	var scale bool
	// длина отрезка между двумя засечками
	var width int

	// максимальный процент ошибок не должен быть меньше 1
	if int(c.MaxRate) < 1 {
		scale = true
		width = h.screenW / int(c.MaxRate*h.rate100)
	} else {
		width = h.screenW / int(c.MaxRate)
	}

	// разбивка дискретизации с учетом высоты окна
	var height = h.discretNum
	for h.screenH/height < 1 {
		height /= 2
	}

	return &dataHistogram{
		scale:  scale,
		width:  width,
		height: height,
	}
}

// рисуем ось Y
func (h *histogram) printY(c *types.CountReqErr, data *dataHistogram) {
	var step int

	var p int

	for i, v := range c.Rate {
		// ориентируемся на кол-во имеющихся записей с кол-вом запросов
		if i == c.Num {
			break
		}

		// шаг дискретизации
		if step == h.discretNum/data.height-1 {
			step = 0

			continue
		}

		step++

		// рисуем ось Y
		fmt.Printf("%3d%s├", i, h.discretName)
		// процент ошибок не должен быть меньше 1
		if data.scale {
			p = int(v * h.rate100)
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
			fmt.Printf(" %d%%\n", int(v*h.rate100))
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
func (h *histogram) PrintHistogram(c *types.CountReqErr, data *dataHistogram) {
	fmt.Printf("t (период): %s\n", period)
	// рисуем ось Y
	h.printY(c, data)

	fmt.Printf("    └")

	var maxRate int
	// максимальный процент ошибок не должен быть меньше 1
	if data.scale {
		maxRate = int(c.MaxRate * h.rate100)
	} else {
		maxRate = int(c.MaxRate)
	}

	// рисуем ось X
	printX(data, maxRate)

	fmt.Println("")
	fmt.Println("Расшифровка:")
	fmt.Println("%*100 - значения процента ошибок умноженных на 100")
	fmt.Println("% - значения процента ошибок")
}
