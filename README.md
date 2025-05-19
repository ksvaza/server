# server

## Currently active api endpoints
GET /api/car/:id/latest
GET /api/car/:id/power?mass=10&voltage=10&const=10

POST /api/race/:car_id/start
POST /api/race/:car_id/finish

PUT /api/mqtt/send/*topic
GET /api/mqtt/log
DELETE /api/mqtt/log

## Some curl commands to showcase the use of api endponints
<!-->curl -k -X "GET" https://server.lv/api/car/2/latest?pasw=12&login=admin<!-->
curl -k -X "GET" https://server.lv/api/car/2/power?mass=80&const=3

curl -k -X "PUT" -d "Hello, world!" https://svaza.lv/api/mqtt/send/topic
curl -k -X "GET" https://svaza.lv/api/mqtt/log
curl -k -X "DELETE" https://svaza.lv/api/mqtt/log

curl -k -X "PUT" -d "{ \"Uop\":3600, \"Iop\":327, \"Pop\":8699, \"Uip\":6129, \"Wh\":15356 }" https://svaza/api/mqtt/send/PSU_OUT/2

## MQTT topics
'#' is the car name or id used thoughout the entire database and systems

PSU_OUT/# receives car psu data in json like so "PSU":{ "Uop":3600, "Iop":327, "Pop":8699, "Uip":6129, "Wh":15356 }
GPS_OUT/# receives car gps data in json like so "GPS":{ "Lat":12.351242, "Lon":56.131241, "Spd":14.2 }
ACCEL_OUT/# receives car acceleration data in json like so "ACCEL":{ "X":2.351242, "Y":6.131241, "Z":1.42 }
SUS_OUT/# which receives car system status in data unlike json. examples: SPD: 12.2 or RST: POR or RST: 2
