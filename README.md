# server

## Šobrīd izveidotie API endpointi
GET /api/car/:id/latest
GET /api/car/:id/power?mass=80&const=3

PUT /api/mqtt/send/*topic
GET /api/mqtt/log
DELETE /api/mqtt/log

## Dažu curl komandu prototipi atbilsoši requestu tipiem
curl -k -X "GET" https://server.lv/api/car/2/latest
curl -k -X "GET" https://server.lv/api/car/2/power?mass=80&const=3

curl -k -X "PUT" -d "Hello, world!" https://svaza.lv/api/mqtt/send/topic
curl -k -X "GET" https://svaza.lv/api/mqtt/log
curl -k -X "DELETE" https://svaza.lv/api/mqtt/log

curl -k -X "PUT" -d "{ \"Uop\":3600, \"Iop\":327, \"Pop\":8699, \"Uip\":6129, \"Wh\":15356 }" https://svaza/api/mqtt/send/PSU_OUT/2

PSU_OUT/12
GPS_OUT/12