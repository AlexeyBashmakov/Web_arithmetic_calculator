/*document.addEventListener("DOMContentLoaded", function(){
    console.log("DOMContentLoaded!")
})*/

// var counter = 0

window.onload = function() {
    console.log("Страница загружена")
    var addButton = document.getElementById("add_button");
    addButton.onclick = handleAddButton;
    var getByIdButton = document.getElementById("get_by_id_button");
    getByIdButton.onclick = handleGetByIdButton;

    setInterval(getListExpression, 5000);
    // setTimeout(getListExpression, 1000);
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
        body: `{"expression": "${expression}"}`
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
        if (response.status === 422) {
            id_expr.innerHTML = "Сервер сообщил о невалидных данных. Возможно ошибка во введённой формуле.";
            expressionInput.value = "";
        }
    }
}

// обработчик кнопки получения выражения по идентификатору
async function handleGetByIdButton(e) {
    e.preventDefault();
    var identificatorInput = document.getElementById("identificator");
    var identificator = identificatorInput.value;
    console.log("identificator:", identificator);
    var path = "/api/v1/expressions/" + identificator;
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
    var path = "/api/v1/expressions";
    var response = await fetch(path, {
        method: "GET",
    });
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
    }
}
