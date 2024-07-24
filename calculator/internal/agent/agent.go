package agent

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"time"

	"web_arithmetic_calculator/internal/constants"
	"web_arithmetic_calculator/internal/environ_vars"
	"web_arithmetic_calculator/pkg/to_log"

	pb "web_arithmetic_calculator/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // для упрощения не будем использовать SSL/TLS аутентификация
)

/*
	структура используется для передачи задач от сервера агенту

и от агента горутинам
*/
type Task_agent struct {
	Id             string  `json:"id"`
	Arg1           float64 `json:"arg1"`
	Arg2           float64 `json:"arg2"`
	Operation      string  `json:"operation"`
	Operation_time float64 `json:"operation_time"`
}

/* структура используется для десериализации сервером данных результата вычисления, полученных в HTTP-запросе агента */
type Result_get struct {
	Id     string  `json:"id"`
	Result float64 `json:"result"`
}

/* Агент
Демон, который получает выражение для вычисления с сервера, вычисляет его и
отправляет на сервер результат выражения.
При старте демон запускает несколько горутин, каждая из которых выступает в роли независимого вычислителя.
Количество горутин регулируется переменной среды COMPUTING_POWER
Агент обязательно общается с оркестратором по http
Агент все время приходит к оркестратору с запросом "дай задачку поработать"
(в ручку GET internal/task для получения задач).
Оркестратор отдаёт задачу.
Агент производит вычисление и в ручку оркестратора (POST internal/task для приёма результатов обработки данных)
отдаёт результат.*/

func Run(d chan bool, logger *slog.Logger) error {
	_exit := make(chan bool)         // канал для передачи горутинам сигнала о завершении работы
	_task := make(chan Task_agent)   // канал для передачи горутинам задачи для расчёта
	_result := make(chan Result_get) // канал для получения от горутин результат расчёта
	// номер порта RPC, на котором работает оркестратор-сервер, получаем из переменной окружения
	port := environ_vars.GetValue(constants.RPC_PORT)
	msg := fmt.Sprintf("Старт агента (используется порт %s)", port)
	to_log.Print(to_log.Param{Msg: msg, Logger: logger})

	host := "localhost"
	addr := fmt.Sprintf("%s:%s", host, port) // используем адрес сервера
	// установим соединение
	// conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))  <- deprecated
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		msg := fmt.Sprintf("Агент: could not connect to grpc server:\n%s\n", err.Error())
		to_log.Print(to_log.Param{Msg: msg, Logger: logger, Level: 1})
		return err
	}

	// создадим клиента для общения с сервером-оркестратором по gRPC
	grpcClient := pb.NewRPCServiceClient(conn)

	g := number_of_goroutines()
	msg = fmt.Sprintf("Старт %d горутин(ы)", g)
	to_log.Print(to_log.Param{Msg: msg, Logger: logger})
	for i := 0; i < g; i++ {
		go evaluator(i, _task, _result, _exit, logger)
	}

	// время, через которое агент запрашивает у сервера задачу
	timeRequ, ok := environ_vars.GetValueInt(constants.TimeRequ)
	if !ok {
		timeRequ = 2000
	}

	go func() {
		// закроем соединение, когда выйдем из функции
		defer conn.Close()
		for {
			select {
			case <-d: // канал для сигнала агенту для завершения работы
				to_log.Print(to_log.Param{Msg: "Агент завершает работу", Logger: logger})
				for i := 0; i < g; i++ {
					_exit <- true
				}
				return
			case <-time.After(time.Duration(timeRequ) * time.Millisecond):
				// Агент все время приходит к оркестратору с запросом "дай задачку поработать"
				resp, err := grpcClient.TaskGet(context.TODO(), &pb.TaskRequest{})
				/* resp - Сообщение, описывающее параметры задачи
				type TaskResponse struct {
					state         protoimpl.MessageState
					sizeCache     protoimpl.SizeCache
					unknownFields protoimpl.UnknownFields

					Id            string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"` // числа здесь - порядок полей в сообщении
					Arg1          float64 `protobuf:"fixed64,2,opt,name=arg1,proto3" json:"arg1,omitempty"`
					Arg2          float64 `protobuf:"fixed64,3,opt,name=arg2,proto3" json:"arg2,omitempty"`
					Operation     string  `protobuf:"bytes,4,opt,name=operation,proto3" json:"operation,omitempty"`
					OperationTime float64 `protobuf:"fixed64,5,opt,name=operation_time,json=operationTime,proto3" json:"operation_time,omitempty"`
				} 				*/
				if err != nil {
					msg := fmt.Sprintf("Агент: %s\n", err.Error())
					to_log.Print(to_log.Param{Msg: msg, Logger: logger, Level: 1})
				} else {
					task := Task_agent{Id: resp.Id,
						Arg1:           resp.Arg1,
						Arg2:           resp.Arg2,
						Operation:      resp.Operation,
						Operation_time: resp.OperationTime}
					// fmt.Println(task)
					go func() {
						_task <- task // передаём задачу горутине
					}()
				}
			case r := <-_result: // получаем от горутины результат задачи
				// Агент производит вычисление и в ручку оркестратора (POST internal/task для приёма результатов
				// обработки данных) отдаёт результат
				fmt.Printf("Агент: ответ от горутины: %v\n", r)
				/* r - структура используется для десериализации сервером данных результата вычисления, полученных в HTTP-запросе агента
				type Result_get struct {
					Id     string  `json:"id"`
					Result float64 `json:"result"`
				} */
				msg := ""
				level := 0
				res := pb.ResultRequest{Id: r.Id, Result: r.Result}
				_, err := grpcClient.TaskResult(context.TODO(), &res)
				if err != nil {
					msg = fmt.Sprintf("Агент: %s", err.Error())
					level = 1
				} else {
					msg = "Агент: сервер успешно получил ответ"
					level = 0
				}
				to_log.Print(to_log.Param{Msg: msg, Logger: logger, Level: level})
			}
		}
	}()

	return nil
}

var empty_task = Task_agent{}

/*
	горутина - вычислитель (evaluator - оценщик потому, что простые выражения считает)

id - номер горутины,
_task - канал для получения задачи для расчёта,
_result - канал для возврата результат расчёта,
_exit - канал для получения сигнала о завершении работы
*/
func evaluator(id int, _task chan Task_agent, _result chan Result_get, _exit chan bool, logger *slog.Logger) {
	msg := fmt.Sprintf("Горутина %d начинает работу", id)
	to_log.Print(to_log.Param{Msg: msg, Logger: logger})
	time_add, ok := environ_vars.GetValueInt(constants.TimeAdd)
	if !ok {
		time_add = 10000
	}
	time_sub, ok := environ_vars.GetValueInt(constants.TimeSub)
	if !ok {
		time_sub = 10000
	}
	time_mult, ok := environ_vars.GetValueInt(constants.TimeMult)
	if !ok {
		time_mult = 10000
	}
	time_div, ok := environ_vars.GetValueInt(constants.TimeDiv)
	if !ok {
		time_div = 10000
	}
	var add = func(a, b float64) float64 {
		return a + b
	}
	var sub = func(a, b float64) float64 {
		return a - b
	}
	var mult = func(a, b float64) float64 {
		return a * b
	}
	var div = func(a, b float64) float64 {
		return a / b
	}
	for {
		select {
		case <-_exit:
			msg = fmt.Sprintf("Горутина %d завершает работу", id)
			to_log.Print(to_log.Param{Msg: msg, Logger: logger})
			return
		case task := <-_task:
			if task == empty_task {
				continue
			}
			msg = fmt.Sprintf("Горутина %d: получена задача: %v", id, task)
			to_log.Print(to_log.Param{Msg: msg, Logger: logger})
			result := Result_get{}
			result.Id = task.Id
			var f func(x, y float64) float64
			var t time.Duration
			switch task.Operation {
			case constants.OPl:
				f = add
				t = time.Duration(time_add) * time.Millisecond
			case constants.OMn:
				f = sub
				t = time.Duration(time_sub) * time.Millisecond
			case constants.OMl:
				f = mult
				t = time.Duration(time_mult) * time.Millisecond
			case constants.ODv:
				f = div
				t = time.Duration(time_div) * time.Millisecond
			}
			result.Result = f(task.Arg1, task.Arg2)
			time.Sleep(t)
			_result <- result
		}
	}
}

// функция получает количество горутин для запуска,
// количество определяется из переменной среды constants.CompPow = "COMPUTING_POWER"
func number_of_goroutines() int {
	goroutines := 1
	g := environ_vars.GetValue(constants.CompPow)
	var err error
	goroutines, err = strconv.Atoi(g)
	if err != nil {
		avail_CPUs := runtime.NumCPU() / 2
		if avail_CPUs <= 0 {
			avail_CPUs = 1
		}
		fmt.Printf("Error on conversion environment variable '%s' from string to int! Setting by default in %d\n", constants.CompPow, avail_CPUs)
		goroutines = avail_CPUs
	}

	return goroutines
}
