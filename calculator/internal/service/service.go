package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"web_arithmetic_calculator/internal/constants"
	"web_arithmetic_calculator/internal/environ_vars"
	"web_arithmetic_calculator/pkg/my_jwt"
	"web_arithmetic_calculator/pkg/my_queue"
	"web_arithmetic_calculator/pkg/proxy_sqlite"
	"web_arithmetic_calculator/pkg/rpn"
)

// состояния задач
const (
	Wait      = "в очереди"
	Calculate = "вычисляется"
	Finished  = "завершено"
)

/*
	описание отдельной задачи

Expression - выражение переданное для расчёта и прошедшее проверку на валидность,
RPN - выражение, переведённое в обратную польскую нотацию
Status - состояние расчёта
Result - результат вычисления
*/
// type Description struct {
// 	Expression string
// 	RPN        []string
// 	Status     string
// 	Result     float64
// }

/*
	структура, содержащая словарь (карту) задач, каждую определяемую своим целочисленным идентификатором,

и потокобезопасную очередь, содержащую задания для расчёта не связанных арифметичеких выражений
*/
// type Calc struct {
// 	Pool  map[int]Description
// 	Queue my_queue.ConcurrentQueue
// }

type Calc struct {
	DB    *proxy_sqlite.Proxy      // БД для хранения пользователей и их задач
	RPN   map[string][]string      // словарь задач для расчёта и их идентификаторов (в виде строки)
	Queue my_queue.ConcurrentQueue // потокобезопасная очередь, содержащая задания для расчёта не связанных арифметичеких выражений
}

// фабрика, создающая экземпляр структуры, управляемой веб-сервисом
func NewCalc(ctx context.Context) (*Calc, error) {
	// return &Calc{Pool: make(map[int]Description), Queue: my_queue.ConcurrentQueue{}}
	db, err := proxy_sqlite.NewProxy(ctx)
	if err != nil {
		return nil, err
	}

	c := Calc{DB: db, RPN: make(map[string][]string), Queue: my_queue.ConcurrentQueue{}}

	/* здесь восстанавливать состояние из БД и наполнять очередь задач
	(1) получаем из БД список выражений
	(2) проходим по списку и выражения, у которых статус не "завершено", помещаем в очередь
	*/
	// es, err := c.DB.SelectExpressionsByUserID(user_id) // <- (1)
	es, err := c.DB.SelectExpressions() // <- (1)
	if err != nil {
		fmt.Printf("Cannot restore state from database:\n%s\n", err.Error())
		return nil, err
	}
	/* es[i] - модель представления арифметического выражения в базе
	Expression struct {
		ID         int64
		Expression string
		Status     string
		Result     float64
		User_id    int64
	}
	*/
	for i := range es {
		if es[i].Status == Finished {
			continue
		}
		rpn_ := rpn.FromInfics(es[i].Expression)
		if (len(rpn_) == 1) && (rpn_[0] == "") { // невалидные данные в базе! откуда?
			fmt.Println("Ошибка перевода из инфиксной записи в ОПН: возможно ошибка во введённой формуле.")
		} else {
			// вставляем ОПН в словарь задач для расчёта и их идентификаторов (в виде строки)
			// ключ данной ОПН в словаре
			k := fmt.Sprintf("%d.%d", es[i].User_id, es[i].ID)
			c.RPN[k] = rpn_
			c.RPN_to_tasks(k) // <- (2)
			fmt.Printf("ОПН\n%v", c.RPN)
		}
	}

	return &c, nil
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
func (c *Calc) RPN_to_tasks(id string) {
	rpn := c.RPN[id]
	for i := range rpn {
		s0 := rpn[i]
		if (s0 == "") || (s0 == ".") {
			continue
		}
		if i+1 == len(rpn) {
			break
		}
		s1 := rpn[i+1]
		if s1 == "." {
			continue
		}
		a := 0
		for s1 == "" {
			a++
			if i+a+1 == len(rpn) {
				return
			}
			s1 = rpn[i+a+1]
		}
		if s1 == "." {
			continue
		}
		if i+a+2 == len(rpn) {
			break
		}
		s2 := rpn[i+a+2]
		if s2 == "." {
			continue
		}
		b := 0
		for s2 == "" {
			b++
			if i+a+b+2 == len(rpn) {
				return
			}
			s2 = rpn[i+a+b+2]
		}
		if s2 == "." {
			continue
		}

		if (s2 == constants.OPl) || (s2 == constants.OMn) || (s2 == constants.OMl) || (s2 == constants.ODv) {
			// если третий символ - операция, то проверяю два предыдущих
			if n0, e0 := strconv.ParseFloat(s0, 64); e0 == nil {
				if n1, e1 := strconv.ParseFloat(s1, 64); e1 == nil {
					// имеем два числа
					// идентификатор задачи формируется из:
					// идентификатора пользователя, идентификатора выражения (метод Calculate) и индекса позиции первого числа данной задачи в ОПН
					task_id := fmt.Sprintf("%s.%d", id, i)
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
					rpn[i] = "."
					rpn[i+a+1] = ""
					rpn[i+a+b+2] = ""
				}
			}
		}

		if i+3 == len(rpn) {
			break
		}
	}
}

// функция проверяет, что данное арифметическое выражение в виде ОПН посчитана
func (c *Calc) RPN_is_finished(id string) bool {
	rpn := c.RPN[id]
	for i := 1; i < len(rpn); i++ {
		if rpn[i] != "" {
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
	fmt.Println("Очередь задач:")
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
	Token string `json:"token"`
	Expr  string `json:"expression"`
}

/*
	Добавление вычисления арифметического выражения

(1) декодируем полученное выражение из json в простой текст
(2) проверяется валидность JWT-токена и что у него не истекло время жизни
(3) проверяется, что в БД такой пользователь существует
(4) проверяется, что у этого пользователя в БД такой JWT-токен
(5) выражение преобразуется из инфиксной записи в обратную польскую нотацию (ОПН)
(6) выражению присваивается идентификатор
(7) полученная ОПН разбивается на независимые задачи для расчёта, которые помещаются в очередь
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
		u_ID, err := c.checkRequestsDatas(task.Token) // <- (2)(3)(4)
		if err != nil {
			fmt.Println(err.Error())
			if err.Error() == "token has invalid claims: token is expired" {
				http.Error(w, err.Error(), http.StatusUnauthorized) // 401
			} else {
				http.Error(w, err.Error(), http.StatusLocked) // 423
			}
			return
		}
		fmt.Printf("От пользователя с токеном\n%s\n", task.Token)
		fmt.Printf("получено выражение: %s\n", task.Expr)
		fmt.Printf("Идентификатор пользователя: %d\n", u_ID)

		// перевод выражения из инфиксной формы в ОПН
		rpn_ := rpn.FromInfics(task.Expr)        // <- (5)
		if (len(rpn_) == 1) && (rpn_[0] == "") { // невалидные данные
			fmt.Println("Ошибка перевода из инфиксной записи в ОПН: возможно ошибка во введённой формуле.")
			http.Error(w, "", http.StatusUnprocessableEntity)
		} else {
			fmt.Println("ОПН:", rpn_)
			id := c.generation_id(u_ID, task.Expr, rpn_) // <- (6)
			k := fmt.Sprintf("%d.%d", u_ID, id)
			c.RPN_to_tasks(k) // <- (7)
			c.show_queue()    // вывод в консоль содержимого очереди
			/* отправляю обратно клиенту в виде json "идентификатор" задачи */
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			// err = json.NewEncoder(w).Encode(map[string]string{"expression": task.Expr})  так я отправлял обратно клиенту полученное выражение в виде json, для проверки
			err = json.NewEncoder(w).Encode(map[string]int64{"id": id}) // возврат клиенту идентификатора присвоенного выражению
			if err != nil {
				fmt.Println("Ошибка кодировки и отправки ответа")
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

	}
}

/*
	функция генерирует идентификатор для арифметического выражения

идентификаторы каждого пользователя будут последовательно увеличиваться начиная от 1
(1) вставляем в БД выражение данного пользователя - получаем идентификатор данного выражения
(2) вставляем ОПН в словарь задач для расчёта и их идентификаторов (в виде строки)
*/
func (c *Calc) generation_id(user_id int64, expr string, rpn []string) int64 {
	expr_to_ins := proxy_sqlite.Expression{Expression: expr, Status: Wait, Result: 0, User_id: user_id}
	// вставляем в БД выражение данного пользователя - получаем идентификатор данного выражения
	last_id, err := c.DB.InsertExpression(&expr_to_ins)
	if err != nil {
		fmt.Printf("Ошибка обращения к базе данных!\n%s", err.Error())
		return -1
	}

	fmt.Printf("Выражению присвоен идентификатор: %d\n", last_id)
	// вставляем ОПН в словарь задач для расчёта и их идентификаторов (в виде строки)
	// ключ данной ОПН в словаре
	k := fmt.Sprintf("%d.%d", user_id, last_id)
	c.RPN[k] = rpn
	return last_id
}

// структура для передачи в json клиенту статуса выражения по его идентификатору
type expr struct {
	Id         int64   `json:"id"`
	Expression string  `json:"expression"`
	Status     string  `json:"status"`
	Result     float64 `json:"result"`
}

// структура для передачи в json клиенту списка выражений
type exprs struct {
	Id     int64   `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

/*
	Получение выражения по его идентификатору

https://habr.com/ru/companies/avito/articles/805097/

(0) получаем идентификатор выражения из пути
(1) получаем JWT-токен из запроса
(2) проверяется валидность JWT-токена и что у него не истекло время жизни
(3) проверяется, что в БД такой пользователь существует
(4) проверяется, что у этого пользователя в БД такой JWT-токен
(5) запрос в БД выражения
*/
func (c *Calc) Expression_id(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	idString := req.PathValue("id") // <- (0), строка
	if path != ("/api/v1/expressions/" + idString) {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	id, err := strconv.Atoi(idString) // <- (0), число
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	token := strings.Split(req.RequestURI, "=")[1] // <- (1)
	u_ID, err := c.checkRequestsDatas(token)       // <- (2)(3)(4)
	if err != nil {
		fmt.Println(err.Error())
		if err.Error() == "token has invalid claims: token is expired" {
			http.Error(w, err.Error(), http.StatusUnauthorized) // 401
		} else {
			http.Error(w, err.Error(), http.StatusLocked) // 423
		}
		return
	}
	fmt.Printf("От пользователя с токеном\n%s\n", token)
	fmt.Println("Получен запрос на выражение с идентификатором", idString)

	task, err := c.DB.SelectExpressionByIDs(int64(id), u_ID) // <- (5)
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("id: %d, expression: %s, status: %s, result: %.2f, user_id: %d\n", task.ID, task.Expression, task.Status, task.Result, task.User_id)
	a := expr{Id: task.ID, Expression: task.Expression, Status: task.Status, Result: task.Result}
	err = json.NewEncoder(w).Encode(map[string]expr{"expression": a})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

(1) получаем JWT-токен из запроса
(2) проверяется валидность JWT-токена и что у него не истекло время жизни
(3) проверяется, что в БД такой пользователь существует
(4) проверяется, что у этого пользователя в БД такой JWT-токен
(5) запрос в БД списка выражений
*/
func (c *Calc) Expressions(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/api/v1/expressions" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	token := strings.Split(req.RequestURI, "=")[1] // <- (1)
	u_ID, err := c.checkRequestsDatas(token)       // <- (2)(3)(4)
	if err != nil {
		fmt.Println(err.Error())
		if err.Error() == "token has invalid claims: token is expired" {
			http.Error(w, err.Error(), http.StatusUnauthorized) // 401
		} else {
			http.Error(w, err.Error(), http.StatusLocked) // 423
		}
		return
	}
	fmt.Printf("От пользователя с токеном\n%s\n", token)
	fmt.Println("Получен запрос на список выражений")

	t := make([]exprs, 0)
	a := exprs{}
	es, err := c.DB.SelectExpressionsByUserID(u_ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := range es {
		a.Id = es[i].ID
		a.Status = es[i].Status
		a.Result = es[i].Result
		t = append(t, a)
	}
	fmt.Println(es)
	r := map[string][]exprs{"expressions": t}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type (
	authenticationDatas struct {
		Username string `json:"username"`
		Passwd   string `json:"passwd"`
	}

	// структура для возврата пользователю JWT-токена
	token_answer struct {
		Token string `json:"token"`
	}
)

/*
	регистрация пользователя в системе

(1) декодирует запрос с помощью json, получает Username, Passwd
(2) запрос в БД на существование пользователя с таким Username
(3) если пользователя с таким Username в БД нет, то генерируем хеш от пароля, JWT-токен и записываем это в БД

	(может добавить в переменные окружения время жизни JWT-токена? - для демонстрации аутентификации по токену и логину),

(4)	пользователю возвратить JWT-токен и перейти в его личный кабинет
*/
func (c *Calc) Registration(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/api/v1/register" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	var query authenticationDatas
	// fmt.Printf("Request:\n%v\n", req.Body)
	err := json.NewDecoder(req.Body).Decode(&query) // <- (1)
	defer req.Body.Close()
	if err != nil { // что-то пошло не так
		fmt.Println(err.Error()) // "The received data is an invalid JSON encoding"
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	fmt.Printf("Получен запрос на регистрацию:\nимя(логин): %s\nпароль: %s\n", query.Username, query.Passwd)
	u, err := c.DB.SelectUserByLogin(query.Username) // <-(2)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// такого пользователя ещё не зарегистрировано
			fmt.Println("такого пользователя ещё не зарегистрировано")
			h := my_jwt.GenerateHash(query.Passwd)             // <- (3), генерируем хеш от пароля
			jwt, err := my_jwt.CreateJWT_token(query.Username) // <- (3), генерируем JWT-токен
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, "", http.StatusInternalServerError)
				return
			} else {
				fmt.Printf("%s\n%s\n", h, jwt)
				id, err := c.DB.InsertUser(&proxy_sqlite.User{
					Login:  query.Username,
					Passwd: h,
					Token:  jwt,
				}) // <- (3), вставляем пользователя в БД
				if err != nil {
					fmt.Println(err.Error())
					http.Error(w, "", http.StatusInternalServerError)
					return
				} else {
					// пользователь создан, надо вернуть пользователю его JWT
					// после получения, которого он должен перейти в свой личный кабинет
					fmt.Printf("пользователь добавлен в БД, id: %d\n", id)
					w.WriteHeader(http.StatusCreated)
					a := token_answer{Token: jwt}
					err = json.NewEncoder(w).Encode(map[string]token_answer{"token": a}) // <- (4)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					// fmt.Fprintf(w, "")
					return
				}
			}
		} else {
			fmt.Println(err.Error()) // "The received data is an invalid JSON encoding"
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	} else { // пользователь есть, сообщить, что такой Username занят
		fmt.Printf("Пользователь с таким именем в БД существует:\nid = %d\nlogin = %s\npasswd = %s\ntoken = %s\n", u.ID, u.Login, u.Passwd, u.Token)
		fmt.Println(u)
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "")
	}
}

/*
	Аутентификация пользователя в системе

(1) декодирует запрос с помощью json, получает Username, Passwd
(2) запрос в БД на существование пользователя с таким Username
(3) если пользователь с таким Username в БД есть, то проверяем пароль
(4) проверяем время жизни JWT-токена, при истечении которой, генерируем новый и записываем его в БД

	(может добавить в переменные окружения время жизни JWT-токена? - для демонстрации аутентификации по токену и логину),

(5)	пользователю возвратить JWT-токен и перейти в его личный кабинет
*/
func (c *Calc) Login(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path != "/api/v1/login" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	var query authenticationDatas
	err := json.NewDecoder(req.Body).Decode(&query) // <- (1)
	defer req.Body.Close()
	if err != nil { // что-то пошло не так
		fmt.Println(err.Error()) // "The received data is an invalid JSON encoding"
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	fmt.Printf("Получен запрос на аутентификацию:\nимя(логин): %s\nпароль: %s\n", query.Username, query.Passwd)
	u, err := c.DB.SelectUserByLogin(query.Username) // <- (2)
	if (err != nil) && (err.Error() == "sql: no rows in result set") {
		fmt.Println(err.Error()) // такой пользователь не зарегистрирован
		http.Error(w, "", http.StatusNotFound)
		return
	}
	h := my_jwt.GenerateHash(query.Passwd)   // генерируем хеш введенного пароля
	if !my_jwt.HashesAreEqual(h, u.Passwd) { // <- (3), проверка хешей паролей
		fmt.Println(errors.New("passwords doesn't matches")) // пароли не совпали
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	u.Passwd = h
	if _, err = my_jwt.CheckJWT_token(u.Token); err != nil { // <- (4), проверяем время жизни JWT-токена
		if err.Error() == "token has invalid claims: token is expired" { // <- время жизни JWT-токена истекло
			fmt.Printf("Время жизни JWT-токена пользователя %s истекло\nГенерируется новый\n", u.Login)
			jwt, err := my_jwt.CreateJWT_token(query.Username) // <- (4), генерируем JWT-токен
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			u.Token = jwt
			err = c.DB.UpdateUser(&u) // <- (4), вставляем JWT-токен в БД
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		} else {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// пользователь прошёл аутентификацию, надо вернуть пользователю его JWT
	// после получения, которого он должен перейти в свой личный кабинет
	fmt.Printf("пользователь: %v\n", u)
	a := token_answer{Token: u.Token}
	err = json.NewEncoder(w).Encode(map[string]token_answer{"token": a}) // <- (5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
	функция проверяет, что данные запроса - имя пользователя и JWT-токен валидны

(1) проверяется валидность JWT-токена и что у него не истекло время жизни
(2) проверяется, что в БД такой пользователь существует
(3) проверяется, что у этого пользователя в БД такой JWT-токен
*/
func (c *Calc) checkRequestsDatas(token string) (int64, error) {
	username := ""
	if u, err := my_jwt.CheckJWT_token(token); err != nil { // <- (1)
		// token has invalid claims: token is expired
		return -1, err
	} else {
		username = u
	}
	u, err := c.DB.SelectUserByLogin(username) // <- (2)
	if (err != nil) && (err.Error() == "sql: no rows in result set") {
		return -1, err
	}
	if u.Token != token { // <- (3)
		err := errors.New("invalid token in request")
		return -1, err
	}

	return u.ID, nil
}
