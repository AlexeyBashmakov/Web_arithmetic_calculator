// var counter = 0

window.onload = function() {
    console.log("Страница загружена")
    var loginButton = document.getElementById("login_button");
    loginButton.onclick = handleLoginButton;
    var registerButton = document.getElementById("register_button");
    registerButton.onclick = handleRegisterButton;
}

// обработчик кнопки входа пользователя
async function handleLoginButton(e) {
    e.preventDefault();
    var usernameInput = document.getElementById("login_username");
    var username = usernameInput.value;
    console.log("username:", username);
    var passwordInput = document.getElementById("login_password");
    var password = passwordInput.value;
    console.log("password:", password);
    var response = await fetch("/api/v1/login", {
        method: "POST",
        // для отправки данных в виде json
        body: `{"username": "${username}", "passwd": "${password}"}`
    });
    if (response.ok) {
        var response_json = await response.json();
        console.log(response_json);
        if (response.status === 200) { // StatusOK
            // пользователь прошёл аутентификацию, получение JWT и переход в личный кабинет
            console.log(`token: ${response_json.token.token}`);
            window.location.href = `/lk.html?username=${username}&token=${response_json.token.token}`;
        }
    } else {
        // console.log("Ответ не получен")
        if (response.status === 401) {  // StatusUnauthorized
            alert("Введён неправильный пароль!");
            passwordInput.value = "";
        }
        if (response.status === 404) { // StatusNotFound
            alert("Пользователь с таким именем (логином) не зарегистрирован!");
            passwordInput.value = "";
            usernameInput.value = "";
        }
        if (response.status === 500) {  // StatusInternalServerError
            alert("Внутренняя ошибка сервера");
            passwordInput.value = "";
        }
    }
}

// обработчик кнопки регистрации пользователя
async function handleRegisterButton(e) {
    e.preventDefault();
    var usernameInput = document.getElementById("register_username");
    var username = usernameInput.value;
    console.log("username:", username);
    var passwdInput0 = document.getElementById("register_password0");
    var passwd0 = passwdInput0.value;
    console.log("password:", passwd0);
    var passwdInput1 = document.getElementById("register_password1");
    var passwd1 = passwdInput1.value;
    console.log("password:", passwd1);
    if (passwd0 == "") {
        alert("Пароль не должен быть пустым!");
    } else {
        if (passwd0 != passwd1) {
            passwdInput0.value = "";
            passwdInput1.value = "";
            alert("Пароли не совпадают!");
        } else {
            /* регистрация: запись пользователя в базу и переход в личный кабинет
            https://codex.so/jwt
            клиенту возвращается токен jwt, в базу заносится пара токен-id
            при следующих запросах клиент передает этот токен, а сервер ищет в базе запись
            если запись найдена - пользователя аутентифицируют
            для большей безопасности токену дают определенное время жизни, после 
            которого он становится недействителен
            https://ru.hexlet.io/courses/go-web-development/lessons/auth/theory_unit */

            var response = await fetch("/api/v1/register", {
                method: "POST",
                // для отправки данных в виде json
                body: `{"username": "${username}", "passwd": "${passwd0}"}`
            });
            console.log("response status:", response.status);
            if (response.ok) {
                var response_json = await response.json();
                console.log(response_json);
                if (response.status === 201) {
                    // пользователь создан, получение JWT и переход в личный кабинет
                    console.log(`token: ${response_json.token.token}`);
                    window.location.href = `/lk.html?username=${username}&token=${response_json.token.token}`;
                }
            } else {
                if (response.status === 422) {
                    alert("Сервер сообщил о невалидных данных. Возможно ошибка во введённой формуле.");
                }
                if (response.status === 409) {
                    alert("Пользователь с таким именем (логином) уже зарегистрирован!");
                    passwdInput0.value = "";
                    passwdInput1.value = "";
                    usernameInput.value = "";
                }
            }
        }
    }
}
