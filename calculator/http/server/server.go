package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"web_calculator/http/server/handler"
	"web_calculator/internal/environ_vars"
	"web_calculator/internal/service"
)

// создание маршрутизатора
func new(calcService service.Calc) (http.Handler, error) {
	m, err := handler.New(calcService)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func Run(logger *slog.Logger) (func(context.Context) error, error) {
	// создаём экземпляр объекта, который будет хранить словарь задач для расчёта и управлять этими задачами
	calcService := service.NewCalc()
	m, err := new(*calcService)
	if err != nil {
		return nil, err
	}

	port := fmt.Sprintf(":%s", environ_vars.GetValue("PORT"))
	msg := fmt.Sprintf("Запуск сервера на порту %s", port)
	fmt.Printf("%s...", msg)
	logger.Info(msg)
	// Начиная с версии 1.8 в Go есть возможность корректно завершать работу HTTP-сервера вызовом метода Shutdown().
	// https://tproger.ru/translations/go-web-server?ysclid=lwqqie5cje972900801
	srv := &http.Server{Addr: port, Handler: m}
	go func() {
		// Запускаем сервер в горутине
		if err := srv.ListenAndServe(); err != nil {
			fmt.Println(err.Error())
		}
	}()
	fmt.Println("ОК")

	// вернем функцию для завершения работы сервера
	return srv.Shutdown, nil
}

/*
func h(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Запрос %s методом %s\n", r.URL.String(), r.Method)
		if name == "id" {
			idString := r.PathValue("id")
			fmt.Fprintf(w, "%s: Вы вызвали %s методом %s c id = %s\n", name, r.URL.String(), r.Method, idString)
		} else {
			fmt.Fprintf(w, "%s: Вы вызвали %s методом %s\n", name, r.URL.String(), r.Method)
		}
	}
}

func index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Привет")
	}
}
*/
