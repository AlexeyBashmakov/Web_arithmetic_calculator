package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"web_arithmetic_calculator/http/server/handler"
	"web_arithmetic_calculator/internal/environ_vars"
	"web_arithmetic_calculator/internal/service"
	"web_arithmetic_calculator/pkg/to_log"
	pb "web_arithmetic_calculator/proto"

	"google.golang.org/grpc"
)

// создание маршрутизатора
func new(calcService *service.Calc) (http.Handler, error) {
	m, err := handler.New(calcService)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func runHTTP(calcService *service.Calc, logger *slog.Logger) (func(context.Context) error, error) {
	m, err := new(calcService)
	if err != nil {
		return nil, err
	}

	port := fmt.Sprintf(":%s", environ_vars.GetValue("HTTP_PORT"))
	msg := fmt.Sprintf("Запуск HTTP-сервера на порту %s", port)
	fmt.Printf("%s ...", msg)
	logger.Info(msg)
	// Начиная с версии 1.8 в Go есть возможность корректно завершать работу HTTP-сервера вызовом метода Shutdown().
	// https://tproger.ru/translations/go-web-server?ysclid=lwqqie5cje972900801
	srv := &http.Server{Addr: port, Handler: m}
	go func() {
		defer calcService.DB.Close()
		// Запускаем сервер в горутине
		if err := srv.ListenAndServe(); err != nil {
			fmt.Println(err.Error())
		}
	}()
	fmt.Println("ОК")

	return srv.Shutdown, nil
}

func Run(ctx context.Context, logger *slog.Logger) (func(context.Context) error, error) {
	// создаём экземпляр объекта, который будет хранить задачи для расчёта и управлять этими задачами
	calcService, err := service.NewCalc(ctx)
	if err != nil {
		return nil, err
	}

	err = runRPC(calcService, logger)
	if err != nil {
		return nil, err
	}

	srv_Shutdown, err := runHTTP(calcService, logger)
	if err != nil {
		return nil, err
	}

	// вернем функцию для завершения работы сервера
	return srv_Shutdown, nil
}

type RPCServer struct {
	pb.RPCServiceServer // сервис из сгенерированного пакета
	CalcService         *service.Calc
}

func NewRPCServer(calcService *service.Calc) *RPCServer {
	return &RPCServer{CalcService: calcService}
}

/*
	TaskRequest - пустое сообщение для запроса задачи

Сообщение, описывающее параметры задачи

	type TaskResponse struct {
		state         protoimpl.MessageState
		sizeCache     protoimpl.SizeCache
		unknownFields protoimpl.UnknownFields

		Id            string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"` // числа здесь - порядок полей в сообщении
		Arg1          float64 `protobuf:"fixed64,2,opt,name=arg1,proto3" json:"arg1,omitempty"`
		Arg2          float64 `protobuf:"fixed64,3,opt,name=arg2,proto3" json:"arg2,omitempty"`
		Operation     string  `protobuf:"bytes,4,opt,name=operation,proto3" json:"operation,omitempty"`
		OperationTime float64 `protobuf:"fixed64,5,opt,name=operation_time,json=operationTime,proto3" json:"operation_time,omitempty"`
	}
*/
func (r *RPCServer) TaskGet(ctx context.Context, empty *pb.TaskRequest) (*pb.TaskResponse, error) {
	if r.CalcService.Queue.Len() == 0 { // если очередь задач пуста
		fmt.Println("Очередь задач пуста")
		return &pb.TaskResponse{}, nil
	}
	if task, ok := r.CalcService.Queue.Dequeue().(service.Task_agent); ok {
		fmt.Println("Отправка агенту задачи для расчёта")
		/* изменяем статус ОПН с "в очереди" на "вычисляется"
		ID задачи сейчас имеет вид "1.6.9" и состоит из трёх частей:
		1 - идентификатор пользователя
		6 - номер ОПН,
		9 - позиция в ОПН, куда будет вставлен результат вычисления данной задачи*/
		ids := strings.Split(task.Id, ".")
		if len(ids) == 1 {
			msg := "Ошибка в идентификаторе задачи"
			fmt.Println("RPC-сервер:", msg)
			return &pb.TaskResponse{}, fmt.Errorf(msg)
		}
		user_id_s, expr_id_s := ids[0], ids[1]
		user_id, err := strconv.Atoi(user_id_s)
		if err != nil {
			msg := "Ошибка получения номера ОПН из номера задачи"
			fmt.Println("RPC-сервер:", msg)
			return &pb.TaskResponse{}, fmt.Errorf(msg)
		}
		expr_id, err := strconv.Atoi(expr_id_s)
		if err != nil {
			msg := "Ошибка получения номера ОПН из номера задачи"
			fmt.Println("RPC-сервер:", msg)
			return &pb.TaskResponse{}, fmt.Errorf(msg)
		}
		/* (1) запрос выражения в БД по Id пользователя и выражения
		(2) изменить статус выражения с "в очереди" на "вычисляется"
		(3) записать изменения в БД */
		task_from_db, err := r.CalcService.DB.SelectExpressionByIDs(int64(expr_id), int64(user_id)) // <- (1)
		if err != nil {
			msg := fmt.Sprintf("Ошибка получения выражения из базы данных!\n%s\n", err.Error())
			fmt.Println("RPC-сервер:", msg)
			return &pb.TaskResponse{}, fmt.Errorf(msg)
		}
		task_from_db.Status = service.Calculate // <- (2)
		// fmt.Println(task_from_db)
		err = r.CalcService.DB.UpdateExpression(&task_from_db) // <- (3)
		if err != nil {
			msg := fmt.Sprintf("Ошибка обновления выражения в базе данных!\n%s\n", err.Error())
			fmt.Println("RPC-сервер:", msg)
			return &pb.TaskResponse{}, fmt.Errorf(msg)
		}
		return &pb.TaskResponse{Id: task.Id,
			Arg1:          task.Arg1,
			Arg2:          task.Arg2,
			Operation:     task.Operation,
			OperationTime: task.Operation_time}, nil
	} else {
		fmt.Println("RPC-сервер: задача не найдена")
		return &pb.TaskResponse{}, nil
	}
}

/*
	Сообщение для описания результата вычисления

	type ResultRequest struct {
		state         protoimpl.MessageState
		sizeCache     protoimpl.SizeCache
		unknownFields protoimpl.UnknownFields

		Id     string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
		Result float64 `protobuf:"fixed64,2,opt,name=result,proto3" json:"result,omitempty"`
	}

ResultResponse - пустое сообщение для отправления результата
*/
func (r *RPCServer) TaskResult(ctx context.Context, res *pb.ResultRequest) (*pb.ResultResponse, error) {
	fmt.Printf("Сервер, получен результат: id: %s, result = %.2f\n", res.Id, res.Result)
	/* ID задачи сейчас имеет вид "1.6.9" и состоит из трёх частей:
	1 - идентификатор пользователя
	6 - номер ОПН,
	9 - позиция в ОПН, куда будет вставлен результат вычисления данной задачи*/
	ids := strings.Split(res.Id, ".")
	if len(ids) == 1 {
		msg := "Ошибка в идентификаторе задачи!"
		fmt.Println("RPC-сервер:", msg)
		return nil, fmt.Errorf(msg)
	}
	_u, _i, _p := ids[0], ids[1], ids[2]
	user_id, err := strconv.Atoi(_u)
	if err != nil {
		msg := "Ошибка получения идентификатора пользователя из номера задачи"
		fmt.Println("RPC-сервер:", msg)
		return nil, fmt.Errorf(msg)
	}
	expr_id, err := strconv.Atoi(_i)
	if err != nil {
		msg := "Ошибка получения номера ОПН из номера задачи"
		fmt.Println("RPC-сервер:", msg)
		return nil, fmt.Errorf(msg)
	}
	p, err := strconv.Atoi(_p)
	if err != nil {
		msg := "Ошибка получения позиции результата из номера задачи"
		fmt.Println("RPC-сервер:", msg)
		return nil, fmt.Errorf(msg)
	}
	k := fmt.Sprintf("%d.%d", user_id, expr_id)
	r.CalcService.RPN[k][p] = fmt.Sprintf("%f", res.Result)
	if r.CalcService.RPN_is_finished(k) {
		task_from_db, err := r.CalcService.DB.SelectExpressionByIDs(int64(expr_id), int64(user_id))
		if err != nil {
			msg := fmt.Sprintf("Ошибка получения выражения из базы данных!\n%s\n", err.Error())
			fmt.Println("RPC-сервер:", msg)
			return nil, fmt.Errorf(msg)
		}
		task_from_db.Status = service.Finished
		task_from_db.Result = res.Result
		err = r.CalcService.DB.UpdateExpression(&task_from_db)
		if err != nil {
			msg := fmt.Sprintf("Ошибка обновления выражения в базе данных!\n%s\n", err.Error())
			fmt.Println("RPC-сервер:", msg)
			return nil, fmt.Errorf(msg)
		}
	} else {
		// перестройка ОПН
		r.CalcService.RPN_to_tasks(k)
	}

	return nil, nil
}

func runRPC(calcService *service.Calc, logger *slog.Logger) error {
	host := "localhost"
	port := fmt.Sprintf(":%s", environ_vars.GetValue("RPC_PORT"))
	msg := fmt.Sprintf("Запуск RPC-сервера на порту %s", port)
	fmt.Printf("%s ...", msg)
	logger.Info(msg)

	addr := fmt.Sprintf("%s%s", host, port)
	lis, err := net.Listen("tcp", addr) // будем ждать запросы по этому адресу

	if err != nil {
		msg := fmt.Sprintf("Агент: error starting tcp listener:\n%s\n", err.Error())
		to_log.Print(to_log.Param{Msg: msg, Logger: logger, Level: 1})
		return err
	}

	fmt.Println("ОК")

	// создадим сервер grpc
	grpcServer := grpc.NewServer()
	// объект структуры, которая содержит реализацию
	// серверной части GeometryService
	rpcServiceServer := NewRPCServer(calcService)
	// зарегистрируем нашу реализацию сервера
	pb.RegisterRPCServiceServer(grpcServer, rpcServiceServer)
	// запустим grpc сервер
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			msg := fmt.Sprintf("Агент: error serving grpc:\n%s\n", err.Error())
			to_log.Print(to_log.Param{Msg: msg, Logger: logger, Level: 1})
		}
	}()

	return nil
}
