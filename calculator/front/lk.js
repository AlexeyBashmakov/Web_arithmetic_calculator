/*document.addEventListener("DOMContentLoaded", function(){
    console.log("DOMContentLoaded!")
})*/

// var counter = 0
var User = {
    Name: "",
    Token: ""
};

var timerId;

window.onload = function() {
    console.log("Страница загружена")
    // нужно получить имя пользователя из строки запроса и его JWT-токен оттуда же
    var param = parseQuerystring(window.location.href.split("?")[1]);
    if (param.length === 0) {
        alert("Не получены параметры пользователя!");
        return;
    }
    User.Name = param[0];
    User.Token = param[1];
    console.log(`user : ${User.Name}`);
    console.log(`token: ${User.Token}`);
    settingUsername();

    var addButton = document.getElementById("add_button");
    addButton.onclick = handleAddButton;
    var getByIdButton = document.getElementById("get_by_id_button");
    getByIdButton.onclick = handleGetByIdButton;

    timerId = setInterval(getListExpression, 5000);
    // setTimeout(getListExpression, 1000);

    var headerH = document.getElementById("username").offsetHeight;
    var mainId = document.getElementById("container");
    var bodyH = document.body.offsetHeight;
    mainId.style.offsetHeight = bodyH - headerH + "px";
    console.log(bodyH + ", " + headerH + ", " + mainId.style.offsetHeight);
    // console.log(document.body.getAttribute("style"));
    // console.log(document.getElementById("username").style.height);
    // console.log(document.getElementById("username").style.background);
}

// обработчик кнопки добавления выражения для вычисления
async function handleAddButton(e) {
    e.preventDefault();
    var expressionInput = document.getElementById("expression");
    var expression = expressionInput.value;
    console.log("expression:", expression);
    var response = await fetch("/api/v1/calculate", {
        method: "POST",
        // отправка данных в виде формы
        //body: new FormData(document.querySelector("form"))
        // для отправки данных в виде json
        body: `{"expression": "${expression}", "token": "${User.Token}"}`
    });
    console.log("response status:", response.status);
    var id_expr = document.getElementById("identifier_of_expression");
    if (response.ok) {
        var response_json = await response.json();
        console.log(response_json);
        id_expr.innerHTML = response_json.id;
        expressionInput.value = "";
    } else {
        console.log("Ответ не получен")
        if (response.status === 401) { // StatusUnauthorized
            // истекло время жизни токена, переход на страницу входа в систему
            alert("Истекло время жизни JWT-токена пользователя. Необходим повторный вход в систему.");
            window.location.href = "/";
            // return;
        }
        if (response.status === 423) { // StatusLocked
            id_expr.innerHTML = "Сервер сообщил, что по данным пользователя ничего не найдено.";
            expressionInput.value = "";
            return;
        }
        if (response.status === 422) {
            id_expr.innerHTML = "Сервер сообщил о невалидных данных. Возможно ошибка во введённой формуле.";
            expressionInput.value = "";
            return;
        }
    }
}

// обработчик кнопки получения выражения по идентификатору
async function handleGetByIdButton(e) {
    e.preventDefault();
    var identificatorInput = document.getElementById("identificator");
    var identificator = identificatorInput.value;
    console.log("identificator:", identificator);
    var path = `/api/v1/expressions/${identificator}?token=${User.Token}`;
    console.log("path: " + path);
    var response = await fetch(path, {
        method: "GET",
    });
    console.log("response status:", response.status);
    var expr_by_id = document.getElementById("expression_by_id");
    if (response.ok) {
        var response_json = await response.json();
        console.log(response_json);
        expr_by_id.innerHTML = htmlForExpressionById(response_json);
        identificatorInput.value = "";
    } else {
        console.log("Ответ не получен");
        if (response.status === 401) { // StatusUnauthorized
            // истекло время жизни токена, переход на страницу входа в систему
            alert("Истекло время жизни JWT-токена пользователя. Необходим повторный вход в систему.");
            window.location.href = "/";
            // return;
        }
        if (response.status === 423) { // StatusLocked
            id_expr.innerHTML = "Сервер сообщил, что по данным пользователя ничего не найдено.";
            identificatorInput.value = "";
            return;
        }
        if (response.status === 404) {
            expr_by_id.innerHTML = "Сервер сообщил о невалидных данных. Возможно нет такого выражения.";
            identificatorInput.value = "";
        }
    }
}

// функция возвращает HTML-содержимое для json-ответа выражения по идентификатору
function htmlForExpressionById(in_json) {
    var s0 = "Идентификатор: <span class=\"identifier_of_expression\">" + in_json.expression.id + "</span><br>";
    var s1 = "Выражение: <span class=\"type_of_expression\">" + in_json.expression.expression + "</span><br>";
    var s2 = "Статус: " + in_json.expression.status + "<br>";
    var s3 = "Результат: <span class=\"result_of_expression\">" + in_json.expression.result + "</span><br>";

    return s0 + s1 + s2 + s3;
}

// функция каждую секунду получает список выражений
async function getListExpression() {
    // counter++;
    // console.log("Счётчик:" + counter);
    var path = `/api/v1/expressions?token=${User.Token}`;
    var response;
    try {
        response = await fetch(path, {
            method: "GET",
        });
    }
    catch(error) {
        // перехватываем ошибку сети когда остановлен сервер
        if ((error instanceof TypeError) && ((error.message === "Failed to fetch") || (error.message === "NetworkError when attempting to fetch resource."))) {
            clearInterval(timerId);
            console.log(error.name);
            console.log(error.message);
            console.log("Остановка таймера получения списка выражений");
        }
        return;
    }
    console.log("response status:", response.status);
    if (response.ok) {
        var response_json = await response.json();
        var expressionsTable = document.getElementById("expressions-table-body");
        var containsTable = "<tbody>\n";
        for (var i = 0; i < response_json.expressions.length; i++) {
            console.log("id: " + response_json.expressions[i].id + ", статус: " + response_json.expressions[i].status + ", результат: " + response_json.expressions[i].result);
            containsTable += "<tr><td>" + response_json.expressions[i].id + "</td><td>" + response_json.expressions[i].status + "</td><td>" + response_json.expressions[i].result + "</td></tr>\n";
        }
        containsTable += "</tbody>\n";
        expressionsTable.innerHTML = containsTable;
    } else {
        console.log("Ответ не получен");
        if (response.status === 401) { // StatusUnauthorized
            // истекло время жизни токена, переход на страницу входа в систему
            alert("Истекло время жизни JWT-токена пользователя. Необходим повторный вход в систему.");
            clearInterval(timerId);
            console.log("Остановка таймера получения списка выражений");
            window.location.href = "/";
            // return;
        }
        if (response.status === 423) { // StatusLocked
            alert("Сервер сообщил, что по данным пользователя ничего не найдено.");
            clearInterval(timerId);
            console.log("Остановка таймера получения списка выражений");
            return;
        }
    }
}

/* функция разбирает переданную строку (предположительно строку параметров из строки запроса, см. комментарий в теле функции)
на пару: имя пользователя, JWT-токен
и возвращает эту пару как массив */
function parseQuerystring(query_string) {
    // username=olaf&token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjE2MjUzNTEsImlhdCI6MTcyMTYyNTI5MSwibmFtZSI6Im9sYWYifQ.4oxE37OmxCc2pCJis42ceZVBXpm5sAoYkzjtdVBJ_IY
    var s;
    var username = "";
    var token = "";

    for (pair of query_string.split("&")) {
        s = pair.split("=");
        if (s[0] === "username") {
            username = s[1];
        }
        if (s[0] === "token") {
            token = s[1];
        }
    }
    if ((username === "") || (token === "")) { // если хоть один параметр не установлен
        return []; // возвращаем пустой массив
    }
    return [username, token];
}

// функция устанавливает имя пользователя в шапке личного кабинета
function settingUsername() {
    var usernameHeader = document.getElementById("username");
    var header = usernameHeader.innerText;
    console.log(header);
    header += " " + User.Name;
    console.log(header);
    usernameHeader.innerHTML = header;
}
