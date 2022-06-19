trap "rm test;kill 0" EXIT SIGINT

go build ./test.go
./test -p 10001 &
./test -p 10002 &
./test -p 10003 &

sleep 2
echo ">>> start server"

curl "http://localhost:10003/cache/user_cache/Tom" &
curl "http://localhost:10002/cache/user_cache/Tom" &
curl "http://localhost:10001/cache/user_cache/Tom" &

wait

