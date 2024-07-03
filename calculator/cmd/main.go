package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"reflect"

	"web_calculator/http/agent"
	"web_calculator/http/server"
	"web_calculator/internal/environ_vars"
)

func greeting() {
	prnt("Привет, апельсин :)")
	t0 := []string{""}
	fmt.Println(len(t0), reflect.TypeOf(t0))

	var t1 chan bool
	prnt(t1)
	t1 = make(chan bool)
	go func() {
		var x, ok = <-t1
		prnt(x, ok)
	}()
	t1 <- true
	close(t1)
	var x, ok = <-t1
	prnt(x, ok)
	// prnt(<-t1)
	prnt("======================")
}

func prnt(a ...any) {
	fmt.Println(a...)
}

func main() {
	greeting()

	if !environ_vars.CheckEnvironmentVariables() {
		if environ_vars.SetEnvironmentVariables() != nil {
			os.Exit(3)
		}
	}

	// настраиваем журналирование
	logger, logFile, err := setupLogger()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	// запускаем сервер и получаем функцию, которая
	// элегантно завершает работу сервера без прерывания любых активных соединений (https://pkg.go.dev/net/http@go1.22.4#Server.Shutdown)
	ShutdownFunc, err := server.Run(logger)
	if err != nil {
		os.Exit(1)
	}

	// будем прослушивать канал для приёма сигнала прерывания (например комбинации CTRL+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// https://tproger.ru/translations/go-web-server?ysclid=lwqqie5cje972900801
	// После получения сигнала выделим несколько секунд на корректное завершение работы сервера.
	// Попытка корректного завершения
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	_, cancel := context.WithCancel(context.Background())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// создадим канал для сигнала агенту о завершении работы
	d := make(chan bool)
	// запуск агента
	agent.Run(d, logger)

	<-c
	go func() {
		// отправляем агенту сигнал о завершении работы
		d <- true
	}()
	cancel()
	// завершим работу сервера
	ShutdownFunc(ctx)
	// закроем журнальный файл
	logFile.Close()

	os.Exit(0)
}

// настраиваем журналирование
func setupLogger() (*slog.Logger, *os.File, error) {
	// создаем журнальный файл для записи и добавления
	f, err := os.OpenFile("calculator.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}
	// создаем текстовый handler для записи сообщений в файл
	h := slog.NewTextHandler(f, &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo})
	// создаем сам объект управляющий журналированием
	logger := slog.New(h)

	return logger, f, nil
}

/* веб-сервис по расчёту арифметических выражений
для расчёта каждого выражения оное преобразуется в обратную польскую нотацию,
которая позволяет производить расчёты не связанных действий параллельно
*/
