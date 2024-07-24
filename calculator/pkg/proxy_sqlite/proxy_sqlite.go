package proxy_sqlite

import (
	"context"
	"database/sql"

	// _ в начале строки импорта для того, чтобы go не удалял эту строку (наверно)
	_ "github.com/mattn/go-sqlite3" // Подключим драйвер для sql lite
)

type (
	// экземпляр этого типа будет хранить подключение к базе данных SQLite
	Proxy struct {
		Ctx context.Context
		DB  *sql.DB
	}

	// модель представления пользователя в базе
	User struct {
		ID     int64
		Login  string
		Passwd string
		Token  string
	}

	// модель представления арифметического выражения в базе
	Expression struct {
		ID         int64
		Expression string
		Status     string
		Result     float64
		User_id    int64
	}
)

// создание экземпляра прокси
func NewProxy(ctx context.Context) (*Proxy, error) {
	db, err := sql.Open("sqlite3", "store.db") // Подключимся к СУБД
	if err != nil {
		return nil, err
	}

	err = db.PingContext(ctx) // и проверим успешность подключения
	if err != nil {
		return nil, err
	}

	p := Proxy{Ctx: ctx, DB: db}

	// создаём таблицы пользователей и выражений
	err = p.createTables()
	if err != nil {
		p.Close()
		return nil, err
	}

	return &p, nil
}

// метод для закрытия базы данных
func (p *Proxy) Close() {
	p.DB.Close()
}

// Создаем 2 таблицы: users и expressions, в котором будем хранить пользователей и выражения, которые они отправляют на вычисления
func (p *Proxy) createTables() error {
	const (
		usersTable = `
	CREATE TABLE IF NOT EXISTS users(
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		login TEXT NOT NULL UNIQUE,
		passwd TEXT NOT NULL,
		token TEXT NOT NULL
	);`

		expressionsTable = `
	CREATE TABLE IF NOT EXISTS expressions(
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		expression TEXT NOT NULL,
		status TEXT NOT NULL,
		result REAL NOT NULL,
		user_id INTEGER NOT NULL,
	
		FOREIGN KEY (user_id)  REFERENCES users (id)
	);`
	)

	// Поскольку строки мы не вычитываем, то мы используем ExecContext
	if _, err := p.DB.ExecContext(p.Ctx, usersTable); err != nil {
		return err
	}

	if _, err := p.DB.ExecContext(p.Ctx, expressionsTable); err != nil {
		return err
	}

	return nil
}

/*
модель представления пользователя в базе

	User struct {
		ID         int64
		Login      string
		Passwd     string
		Token      string
	}
*/
func (p *Proxy) InsertUser(user *User) (int64, error) {
	var q = "INSERT INTO users (login, passwd, token) values ($1, $2, $3)"
	result, err := p.DB.ExecContext(p.Ctx, q, user.Login, user.Passwd, user.Token)
	if err != nil {
		return 0, err
	}
	// здесь мы получаем идентификатор последней вставленной строки
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *Proxy) SelectUserByID(id int64) (User, error) {
	u := User{}
	var q = "SELECT id, login, passwd, token FROM users WHERE id = $1"
	/* функция QueryRowContext используется, чтобы получать значение одной строки из СУБД.
	Обратите внимание, что если подставить несуществующий в СУБД идентификатор (например 1000), то мы получим ошибку panic: sql: no rows in result set */
	err := p.DB.QueryRowContext(p.Ctx, q, id).Scan(&u.ID, &u.Login, &u.Passwd, &u.Token)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (p *Proxy) SelectUserByLogin(username string) (User, error) {
	u := User{}
	var q = "SELECT id, login, passwd, token FROM users WHERE login = $1"
	/* функция QueryRowContext используется, чтобы получать значение одной строки из СУБД.
	Обратите внимание, что если подставить несуществующий в СУБД идентификатор (например 1000), то мы получим ошибку panic: sql: no rows in result set */
	err := p.DB.QueryRowContext(p.Ctx, q, username).Scan(&u.ID, &u.Login, &u.Passwd, &u.Token)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (p *Proxy) SelectUserByAuthenticationDatas(user *User) (User, error) {
	u := User{}
	var q = "SELECT id, login, passwd, token FROM users WHERE login = $1 and passwd = $2"
	/* функция QueryRowContext используется, чтобы получать значение одной строки из СУБД.
	Обратите внимание, что если подставить несуществующий в СУБД идентификатор (например 1000), то мы получим ошибку panic: sql: no rows in result set */
	err := p.DB.QueryRowContext(p.Ctx, q, user.Login, user.Passwd).Scan(&u.ID, &u.Login, &u.Passwd, &u.Token)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (p *Proxy) SelectUserByToken(user *User) (User, error) {
	u := User{}
	var q = "SELECT id, login, passwd, token FROM users WHERE token = $1"
	/* функция QueryRowContext используется, чтобы получать значение одной строки из СУБД.
	Обратите внимание, что если подставить несуществующий в СУБД идентификатор (например 1000), то мы получим ошибку panic: sql: no rows in result set */
	err := p.DB.QueryRowContext(p.Ctx, q, user.Token).Scan(&u.ID, &u.Login, &u.Passwd, &u.Token)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (p *Proxy) SelectUsers() ([]User, error) {
	var users []User
	var q = "SELECT id, login, passwd, token FROM users"
	// функция QueryContext используется, чтобы получать значения строк из СУБД
	rows, err := p.DB.QueryContext(p.Ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		u := User{}
		err := rows.Scan(&u.ID, &u.Login, &u.Passwd, &u.Token)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func (p *Proxy) UpdateUser(user *User) error {
	var q = "UPDATE users SET login = $1, passwd = $2, token = $3 WHERE id = $4"
	_, err := p.DB.ExecContext(p.Ctx, q, user.Login, user.Passwd, user.Token, user.ID)
	if err != nil {
		return err
	}

	return nil
}

/*
CREATE TABLE IF NOT EXISTS expressions(

	id INTEGER PRIMARY KEY AUTOINCREMENT,
	expression TEXT NOT NULL,
	status TEXT NOT NULL,
	result REAL NOT NULL,
	user_id INTEGER NOT NULL,

	FOREIGN KEY (user_id)  REFERENCES users (id)
*/
func (p *Proxy) InsertExpression(expr *Expression) (int64, error) {
	var q = `
	INSERT INTO expressions (expression, status, result, user_id) values ($1, $2, $3, $4)
	`
	result, err := p.DB.ExecContext(p.Ctx, q, expr.Expression, expr.Status, expr.Result, expr.User_id)
	if err != nil {
		return 0, err
	}
	// здесь мы получаем идентификатор последней вставленной строки
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (p *Proxy) SelectExpressionByIDs(id, user_id int64) (Expression, error) {
	e := Expression{}
	var q = "SELECT id, expression, status, result, user_id FROM expressions WHERE id = $1 and user_id = $2"
	/* функция QueryRowContext используется, чтобы получать значение одной строки из СУБД.
	Обратите внимание, что если подставить несуществующий в СУБД идентификатор (например 1000), то мы получим ошибку panic: sql: no rows in result set */
	err := p.DB.QueryRowContext(p.Ctx, q, id, user_id).Scan(&e.ID, &e.Expression, &e.Status, &e.Result, &e.User_id)
	if err != nil {
		return e, err
	}

	return e, nil
}

func (p *Proxy) SelectExpressions() ([]Expression, error) {
	var expressions []Expression
	var q = "SELECT id, expression, status, result, user_id FROM expressions"
	// функция QueryContext используется, чтобы получать значения строк из СУБД
	rows, err := p.DB.QueryContext(p.Ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Expression{}
		err := rows.Scan(&e.ID, &e.Expression, &e.Status, &e.Result, &e.User_id)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, e)
	}

	return expressions, nil
}

func (p *Proxy) SelectExpressionsByUserID(user_id int64) ([]Expression, error) {
	var expressions []Expression
	var q = "SELECT id, expression, status, result, user_id FROM expressions WHERE user_id = $1"
	// функция QueryContext используется, чтобы получать значения строк из СУБД
	rows, err := p.DB.QueryContext(p.Ctx, q, user_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Expression{}
		err := rows.Scan(&e.ID, &e.Expression, &e.Status, &e.Result, &e.User_id)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, e)
	}

	return expressions, nil
}

func (p *Proxy) UpdateExpression(expression *Expression) error {
	var q = "UPDATE expressions SET expression = $1, status = $2, result = $3 WHERE id = $4"
	// fmt.Println(expression)
	_, err := p.DB.ExecContext(p.Ctx, q, expression.Expression, expression.Status, expression.Result, expression.ID)
	if err != nil {
		return err
	}

	return nil
}
