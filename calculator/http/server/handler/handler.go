package handler

import (
	"net/http"
	"path/filepath"

	"web_calculator/internal/service"
)

func New(calcService service.Calc) (http.Handler, error) {
	m := http.NewServeMux()

	// Добавление вычисления арифметического выражения
	m.HandleFunc("POST /api/v1/calculate", calcService.Calculate)
	// Получение выражения по его идентификатору
	m.HandleFunc("GET /api/v1/expressions/{id}", calcService.Expression_id)
	// Получение списка выражений
	m.HandleFunc("GET /api/v1/expressions", calcService.Expressions)
	// Получение задачи агентом для выполнения
	m.HandleFunc("GET /internal/task", calcService.Task_get)
	// Прием результата обработки данных от агента
	m.HandleFunc("POST /internal/task", calcService.Task_result)

	// Настройка раздачи статических файлов
	// https://tproger.ru/translations/go-web-server?ysclid=lwqqie5cje972900801
	staticPath, _ := filepath.Abs("../front/")
	fs := http.FileServer(http.Dir(staticPath))
	m.Handle("/*", fs)

	return m, nil
}
