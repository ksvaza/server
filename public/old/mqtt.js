
var username;
var password;

function auth() 
{
    username = 'username';
    password = 'password';
}

function send() 
{
    var message = document.getElementById('message').value;
    var topic = document.getElementById('topic').value;

    var xhr = new XMLHttpRequest();
    xhr.open('PUT', '/api/mqtt/send/' + topic, true);
    xhr.send(message);
}

function refreshLog()
{
    var log = document.getElementById('log');
    log.innerHTML = '';

    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/mqtt/log', true);
    xhr.send();
    xhr.onload = function() {
        var textContent = xhr.responseText;
        log.textContent = textContent;
        console.log(textContent);
        //setTimeout(refreshLog, 1000);
    }
}