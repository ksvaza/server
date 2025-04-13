
var username;
var password;
var carID;
var refreshInterval = 2000;

function auth() 
{
    username = 'username';
    password = 'password';
}

function updateInterval()
{
    refreshInterval = document.getElementById('refreshInterval').value;
    console.log(refreshInterval);
}

function refresh()
{
    carID = document.getElementById('carID').value;

    var xhr = new XMLHttpRequest();
    xhr.open('GET', "/api/car/" + carID + "/latest", true);
    xhr.send();
    xhr.onload = function() {
        var textContent = xhr.responseText;

        // parse the json
        var carData = JSON.parse(textContent);
        console.log(carData);

        // get the elements
        var psuUop = document.getElementById('psuUop');
        var psuIop = document.getElementById('psuIop');
        var psuPop = document.getElementById('psuPop');
        var psuUip = document.getElementById('psuUip');
        var psuWh = document.getElementById('psuWh');
        var psuTime = document.getElementById('psuTime');

        var gpsLat = document.getElementById('gpsLat');
        var gpsLon = document.getElementById('gpsLon');
        var gpsSpd = document.getElementById('gpsSpd');
        var gpsTime = document.getElementById('gpsTime');

        var accelX = document.getElementById('accelX');
        var accelY = document.getElementById('accelY');
        var accelZ = document.getElementById('accelZ');
        var accelTime = document.getElementById('accelTime');

        // update the elements
        psuUop.textContent = carData.PSU.Uop;
        psuIop.textContent = carData.PSU.Iop;
        psuPop.textContent = carData.PSU.Pop;
        psuUip.textContent = carData.PSU.Uip;
        psuWh.textContent = carData.PSU.Wh;
        psuTime.textContent = carData.PSU.Time;

        gpsLat.textContent = carData.GPS.Lat;
        gpsLon.textContent = carData.GPS.Lon;
        gpsSpd.textContent = carData.GPS.Spd;
        gpsTime.textContent = carData.GPS.Time;

        accelX.textContent = carData.ACCEL.X;
        accelY.textContent = carData.ACCEL.Y;
        accelZ.textContent = carData.ACCEL.Z;
        accelTime.textContent = carData.ACCEL.Time;

        // call the refresh function again after the refresh interval
        setTimeout(refresh, refreshInterval);
    }
}