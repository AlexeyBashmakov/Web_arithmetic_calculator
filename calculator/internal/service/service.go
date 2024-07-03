package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"web_calculator/internal/constants"
	"web_calculator/internal/environ_vars"
	"web_calculator/pkg/my_queue"
	"web_calculator/pkg/rpn"
)

// состояния задач
const wait = "в очереди"
const calculate = "вычисляется"
const finished = "завершено"

/*
	описание отдельной задачи

Expression - выражение переданное для расчёта и прошедшее проверку на валидность,
RPN - выражение, переведённое в обратную польскую нотацию
Status - состояние расчёта
Result - результат вычисления
*/
type Description struct {
	Expression string
	RPN        []string
	Status     string
	Result     float64
}

/*
	структура, содержащая словарь (карту) задач, каждую определяемую своим целочисленным идентификатором,

и потокобезопасную очередь, содержащую задания для расчёта не связанных арифметичеких выражений
*/
type Calc struct {
	Pool  map[int]Description
	Queue my_queue.ConcurrentQueue
}

// фаблика, создающая экземпляр структуры, управляемой веб-сервисом
func NewCalc() *Calc {
	return &Calc{Pool: make(map[int]Description), Queue: my_queue.ConcurrentQueue{}}
}

// структура используется для передачи задач агенту
type Task_agent struct {
	Id             string  `json:"id"`
	Arg1           float64 `json:"arg1"`
	Arg2           float64 `json:"arg2"`
	Operation      string  `json:"operation"`
	Operation_time float64 `json:"operation_time"`
}

// структура используется для десериализации данных HTTP-запроса агента результата вычисления
type Result_get struct {
	Id     string  `json:"id"`
	Result float64 `json:"result"`
}

// полученную ОПН разбиваем на задачи
func (c *Calc) rpn_to_tasks(id int) {
	for i := range c.Pool[id].RPN {
		s0 := c.Pool[id].RPN[i]
		if (s0 == "") || (s0 == ".") {
			continue
		}
		if i+1 == len(c.Pool[id].RPN) {
			break
		}
		s1 := c.Pool[id].RPN[i+1]
		if s1 == "." {
			continue
		}
		a := 0
		for s1 == "" {
			a++
			if i+a+1 == len(c.Pool[id].RPN) {
				return
			}
			s1 = c.Pool[id].RPN[i+a+1]
		}
		if s1 == "." {
			continue
		}
		if i+a+2 == len(c.Pool[id].RPN) {
			break
		}
		s2 := c.Pool[id].RPN[i+a+2]
		if s2 == "." {
			continue
		}
		b := 0
		for s2 == "" {
			b++
			if i+a+b+2 == len(c.Pool[id].RPN) {
				return
			}
			s2 = c.Pool[id].RPN[i+a+b+2]
		}
		if s2 == "." {
			continue
		}

		if (s2 == constants.OPl) || (s2 == constants.OMn) || (s2 == constants.OMl) || (s2 == constants.ODv) {
			// если третий символ - операция, то проверяю два предыдущих
			if n0, e0 := strconv.ParseFloat(s0, 64); e0 == nil {
				if n1, e1 := strconv.ParseFloat(s1, 64); e1 == nil {
					// имеем два числа
					// идентификатор задачи формируется из идентификатора выражения (метод Calculate) и индекса позиции первого числа данной задачи в ОПН
					task_id := fmt.Sprintf("%d.%d", id, i)
					operation_time := ""
					switch s2 {
					case constants.OPl:
						operation_time = environ_vars.GetValue(constants.TimeAdd)
					case constants.OMn:
						operation_time = environ_vars.GetValue(constants.TimeSub)
					case constants.OMl:
						operation_time = environ_vars.GetValue(constants.TimeMult)
					case constants.ODv:
						operation_time = environ_vars.GetValue(constants.TimeDiv)
					}
					var op_t float64
					if op_t, e0 = strconv.ParseFloat(operation_time, 64); e0 != nil {
						op_t = 1
					}
					c.Queue.Enqueue(Task_agent{Id: task_id, Arg1: n0, Arg2: n1, Operation: s2, Operation_time: op_t})
					c.Pool[id].RPN[i] = "."
					c.Pool[id].RPN[i+a+1] = ""
					c.Pool[id].RPN[i+a+b+2] = ""
				}
			}
		}

		if i+3 == len(c.Pool[id].RPN) {
			break
		}
	}
}

// функция проверяет, что данное арифметическое выражение в виде ОПН посчитана
func (c *Calc) rpn_is_finished(id int) bool {
	for i := 1; i < len(c.Pool[id].RPN); i++ {
		if c.Pool[id].RPN[i] != "" {
			return false
		}
	}

	return true
}

/*
	функция для вывода в консоль содержимого очереди задач для агента

структура используется для передачи задач агенту

	type Task_agent struct {
		Id             string  `json:"id"`
		Arg1           float64 `json:"arg1"`
		Arg2           float64 `json:"arg2"`
		Operation      string  `json:"operation"`
		Operation_time float64 `json:"operation_time"`
	}
*/
func (c *Calc) show_queue() {
	for i := 0; i < c.Queue.Len(); i++ {
		if item, ok := c.Queue.Dequeue().(Task_agent); ok {
			// fmt.Println(item)
			fmt.Printf("id: %s, arg1: %.2f, arg2: %.2f, operation: %s, time: %.2f\n", item.Id, item.Arg1, item.Arg2, item.Operation, item.Operation_time)
			c.Queue.Enqueue(item)
		} else {
			break
		}
	}
}

// структура, в которой передается выражение для расчёта из html
type Task_expr struct {
	Expr string `json:"expression"`
}

/*
	Добавление вычисления арифметического выражения

(1) декодируем полученное выражение из json в простой текст
(2) преобразуем из инфиксной записи в обратную польскую нотацию (ОПН)
(3) присвоение выражению идентификатора
(4) разбиение полученной ОПН на независимые задачи для расчёта
*/
func (c *Calc) Calculate(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/api/v1/calculate" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	var task Task_expr
	// fmt.Printf("Request:\n%v\n", req.Body)
	err := json.NewDecoder(req.Body).Decode(&task) // <- (1)
	defer req.Body.Close()
	if err != nil { // что-то пошло не так
		fmt.Println(err.Error()) // "The received data is an invalid JSON encoding"
		http.Error(w, "", http.StatusInternalServerError)
		return
	} else {
		fmt.Printf("Получено выражение: %s\n", task.Expr)
		// перевод выражения из инфиксной формы в ОПН
		rpn_ := rpn.FromInfics(task.Expr)        // <- (2)
		if (len(rpn_) == 1) && (rpn_[0] == "") { // невалидные данные
			fmt.Println("Ошибка перевода из инфиксной записи в ОПН: возможно ошибка во введённой формуле.")
			http.Error(w, "", http.StatusUnprocessableEntity)
		} else {
			fmt.Println("ОПН:", rpn_)
			id := c.generation_id(task.Expr, rpn_) // <- (3)
			c.rpn_to_tasks(id)                     // <- (4)
			c.show_queue()                         // вывод в консоль содержимого очереди
			/* отправляю обратно клиенту в виде json "идентификатор" задачи, пока фиктивный */
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			// err = json.NewEncoder(w).Encode(map[string]string{"expression": task.Expr})  так я отправлял обратно клиенту полученное выражение в виде json, для проверки
			err = json.NewEncoder(w).Encode(map[string]int{"id": id}) // возврат клиенту идентификатора присвоенного выражению
			if err != nil {
				fmt.Println("Ошибка кодировки и отправки ответа")
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

	}
}

/*
	функция генерирует идентификатор для арифметического выражения

идентификаторы будут последовательно увеличиваться начиная от 1
*/
func (c *Calc) generation_id(expr string, rpn []string) int {
	last_id := 0
	// прохожу по словарю выражений, получаю только ключ - идентификатор выражения
	for id := range c.Pool {
		if id > last_id { // нахожу наибольший идентификатор
			last_id = id
		}
	}
	fmt.Printf("Генерация идентификатора выражения. Последний существующий id: %d\n", last_id)
	if last_id == 0 {
		last_id = 1
	} else {
		last_id++
	}
	t := Description{Expression: expr, RPN: rpn, Status: wait, Result: 0}
	c.Pool[last_id] = t
	return last_id
}

// структура для передачи в json клиенту статуса выражения по его идентификатору
type expr struct {
	Id         int     `json:"id"`
	Expression string  `json:"expression"`
	Status     string  `json:"status"`
	Result     float64 `json:"result"`
}

// структура для передачи в json клиенту списка выражений
type exprs struct {
	Id     int     `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

/*
	Получение выражения по его идентификатору

https://habr.com/ru/companies/avito/articles/805097/
*/
func (c *Calc) Expression_id(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	idString := req.PathValue("id")
	fmt.Println("Получен запрос на выражение с идентификатором", idString)
	if path != ("/api/v1/expressions/" + idString) {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	task, ok := c.Pool[id]
	if !ok {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	a := expr{Id: id, Expression: task.Expression, Status: task.Status, Result: task.Result}
	err = json.NewEncoder(w).Encode(map[string]expr{"expression": a})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// w.WriteHeader(http.StatusCreated)
	// fmt.Fprintf(w, "")
}

/*
Получение списка выражений
Тело ответа
```

	{
	          "expressions": [
	                {
	                      "id": <идентификатор выражения>,
	                      "status": <статус вычисления выражения>,
	                      "result": <результат выражения>
	                },
	                {
	                      "id": <идентификатор выражения>,
	                      "status": <статус вычисления выражения>,
	                      "result": <результат выражения>
	                 }
	            ]
	}

```
*/
func (c *Calc) Expressions(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/api/v1/expressions" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	fmt.Println("Получен запрос на список выражений")
	t := make([]exprs, 0)
	a := exprs{}
	// прохожу по словарю выражений, получаю только ключ - идентификатор выражения
	for i := range c.Pool {
		// fmt.Println(i, c.Pool[i])
		a.Id = i
		a.Status = c.Pool[i].Status
		a.Result = c.Pool[i].Result
		t = append(t, a)
	}
	r := map[string][]exprs{"expressions": t}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
	Отправка агенту задачи для выполнения

Тело ответа
```

	{
	       "task":
	             {
	                   "id": <идентификатор задачи>,
	                   "arg1": <имя первого аргумента>,
	                   "arg2": <имя второго аргумента>,
	                   "operation": <операция>,
	                   "operation_time": <время выполнения операции>
	              }
	}

```
*/
func (c *Calc) Task_get(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/internal/task" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	if c.Queue.Len() == 0 { // если очередь задач пуста
		http.Error(w, "", http.StatusNotFound)
		return
	}
	/*for k, v := range c.Pool { // проходим по всем задачам
		if v.Status != wait { // если задача вычисляется или уже завершена, то пропускаем её
			continue
		}
		// здесь уже разбиение задачи на операции
		fmt.Println(k)
	}
	http.Error(w, "", http.StatusNotFound)*/
	if task, ok := c.Queue.Dequeue().(Task_agent); ok {
		w.Header().Set("Content-Type", "application/json")
		r := map[string]Task_agent{"task": task}
		err := json.NewEncoder(w).Encode(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			// изменяем статус ОПН с "в очереди" на "вычисляется"
			_id := strings.Split(task.Id, ".")[0]
			if id, err := strconv.Atoi(_id); err != nil {
				fmt.Println("Ошибка получения номера ОПН из номера задачи")
			} else {
				// c.Pool[id].Status = calculate - так не работает, потому что ... https://stackoverflow.com/questions/42605337/cannot-assign-to-struct-field-in-a-map
				if d, ok := c.Pool[id]; ok {
					d.Status = calculate
					c.Pool[id] = d
				}
			}
		}
	} else {
		http.Error(w, "", http.StatusNotFound)
	}
}

/*
	Приём от агента результата вычисления операции

структура используется для десериализации данных HTTP-запроса агента результата вычисления

	type Result_get struct {
		Id     string  `json:"id"`
		Result float64 `json:"result"`
	}
*/
func (c *Calc) Task_result(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/internal/task" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	var result Result_get
	err := json.NewDecoder(req.Body).Decode(&result)
	defer req.Body.Close()
	if err != nil { // что-то пошло не так
		fmt.Println(err.Error())
		http.Error(w, "", http.StatusInternalServerError)
		return
	} else {
		fmt.Println("Сервер, получен результат:", result)
		// идентификатор задачи формируется из идентификатора выражения (метод Calculate) и индекса позиции первого числа данной задачи в ОПН
		ids := strings.Split(result.Id, ".")
		if len(ids) == 1 {
			http.Error(w, "", http.StatusUnprocessableEntity)
		} else {
			_i, _p := ids[0], ids[1]
			if id, err := strconv.Atoi(_i); err != nil {
				fmt.Println("Ошибка получения номера ОПН из номера задачи")
			} else {
				if p, err := strconv.Atoi(_p); err != nil {
					fmt.Println("Ошибка получения позиции результата из номера задачи")
				} else {
					c.Pool[id].RPN[p] = fmt.Sprintf("%f", result.Result)
					fmt.Println(c.Pool[id].RPN)
					if c.rpn_is_finished(id) {
						if d, ok := c.Pool[id]; ok {
							d.Status = finished
							d.Result = result.Result
							c.Pool[id] = d
						}
					} else {
						// перестройка ОПН
						c.rpn_to_tasks(id)
					}
				}
			}
			fmt.Fprintf(w, "")
		}
	}
}
