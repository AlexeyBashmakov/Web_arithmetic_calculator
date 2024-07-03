package rpn

import (
	"strings"

	"web_calculator/internal/constants"
	"web_calculator/pkg/my_stack"
)

const space = " "

/*
	перевод выражения в инфиксную форму,
	функция получает на вход строку - арифметическое выражение для вычисления,
	возвращает - слайс (массив) строк, каждая из которых представляет собой элемент исходного выражения, записанного в ОПН,
	если в исходном выражении ошибка, то возвращается слайс единичной длины с нулевым элементом "",
	по мотивам

https://ru.ruwiki.ru/wiki/Обратная_польская_запись
https://habr.com/ru/articles/747178/
*/
func FromInfics(expression string) []string {
	out := make([]string, 0) // выражение в ОПН
	item := ""               // буфер для формирования выходной строки
	opl := rune(constants.OPl[0])
	omn := rune(constants.OMn[0])
	oml := rune(constants.OMl[0])
	odv := rune(constants.ODv[0])
	op_b := rune("("[0]) // открывающая скобка
	cl_b := rune(")"[0]) // закрывающая скобка

	expression = strings.Trim(expression, " ") // удаление ведущих и замыкающих пробелов из строки
	if (expression[0] == constants.OPl[0]) || (expression[0] == constants.OMn[0]) ||
		(expression[0] == constants.OMl[0]) || (expression[0] == constants.ODv[0]) ||
		(expression[len(expression)-1] == constants.OPl[0]) || (expression[len(expression)-1] == constants.OMn[0]) ||
		(expression[len(expression)-1] == constants.OMl[0]) || (expression[len(expression)-1] == constants.ODv[0]) {
		// проверка на ведущие и замыкающие знаки операций - расцениваю это как ошибки
		out = append(out, "")
		return out
	}

	// проверка на равное количество скобок
	op := 0
	cl := 0
	for _, c := range expression {
		if string(c) == "(" {
			op++
		}
		if string(c) == ")" {
			cl++
		}
	}
	if op != cl {
		out = append(out, "")
		return out
	}
	// ====================================

	stck := my_stack.NewMyStack[rune]()
	// fmt.Println("Выражение:", expression)
	for _, c := range expression {
		if c == rune(space[0]) { // пропускаем пробелы
			continue
		}
		switch c {
		case rune("0"[0]), rune("1"[0]), rune("2"[0]), rune("3"[0]), rune("4"[0]),
			rune("5"[0]), rune("6"[0]), rune("7"[0]), rune("8"[0]), rune("9"[0]), rune("."[0]):
			// читаем символы числа, например: 3.14
			item += string(c)
		case opl, omn: // Если символ является бинарной операцией "+" или "-" ...
			if item != "" {
				out = append(out, item) // операнд записываем в ответ
				item = ""
			}
			for stck.Size() > 0 {
				/* достаём из стека и записываем в ответ все операторы из стека пока они больше или
				равны текущему по приоритету или пока стек не опустеет или пока не встретили открывающую скобку в стеке */
				_s := stck.Pop() // верхний элемент стека
				if _s == op_b {
					stck.Push(_s)
					break
				}
				out = append(out, string(_s))
			}
			stck.Push(c) // помещаем операцию в стек
		case oml, odv: // Если символ является бинарной операцией "*" или "/" ...
			if item != "" {
				out = append(out, item) // операнд записываем в ответ
				item = ""
			}
			for stck.Size() > 0 {
				/* достаём из стека и записываем в ответ все операторы из стека пока они больше или
				равны текущему по приоритету или пока стек не опустеет или пока не встретили открывающую скобку в стеке */
				_s := stck.Pop() // верхний элемент стека
				if (_s == opl) || (_s == omn) {
					stck.Push(_s)
					break
				}
				out = append(out, string(_s))
			}
			stck.Push(c) // помещаем операцию в стек
		case op_b: // если встретили открывающую скобку
			stck.Push(c) //кладём её в стек
		case cl_b: // если встретили закрывающую скобку
			if item != "" {
				out = append(out, item) // операнд записываем в ответ
				item = ""
			}
			// записываем в ответ все операторы из стека пока не достанем из стека открывающую скобку
			for stck.Size() > 0 {
				_s := stck.Pop()
				if _s == op_b {
					break
				}
				out = append(out, string(_s))
			}

		default: // другие случаи, т.е. невалидные данные
			out = out[:1]
			out[0] = ""
			return out
		}
	}
	if item != "" {
		out = append(out, item)
	}
	// Когда входная строка закончилась, выталкиваем все символы из стека в выходную строку
	for stck.Size() > 0 {
		_s := stck.Pop()
		if _s == op_b {
			// В стеке должны были остаться только символы операций; если это не так, значит в выражении не согласованы скобки
			out = out[:1]
			out[0] = ""
			return out
		}
		out = append(out, string(_s))
	}
	verify_str := "()"      // "0123456789.+-*/"
	for _, c := range out { // ещё одна проверка на скобки
		if strings.ContainsAny(c, verify_str) {
			out = out[:1]
			out[0] = ""
			return out
		}
	}
	// fmt.Println(out) // выражение в ОПН
	return out // выражение в ОПН
}
