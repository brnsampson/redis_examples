# redis_examples
Some examples of how redis might be used. Meant to help demonstrate what redis is and how to use it.

There are currently 3 examples here that can be used, cacher, queuer, and messenger. You can get each of these with `go get github.com/brnsampson/redis_examples/cacher`, etc.

CACHER:
A simple web application that presets a get and set endpoint. Configure as follows.

In Linux:
$> REDIS_ADDR=<address of your redis instance>
$> CACHE_ADDR=<address to serve on. Usually ':8080' is good>
$> go build src/github.com/brnsampson/redis_examples/cacher
$> ./cacher.exe (or whatever extension it has)

Windows is similar, but in powershell environment variables are defined with slightly different syntax:
$env:REDIS_ADDR = "<address of redis instance>"

To query:
Adding a key: `curl -XPUT 127.0.0.1:8080/set -d '{"animal": "cat"}'`
retrieveing a key: `curl 127.0.0.1:8080/get?key=animal`

QUEUER:
A simple web application that presets a push and pop endpoint. Configure as follows.

In Linux:
$> REDIS_ADDR=<address of your redis instance>
$> QUEUE_ADDR=<address to serve on. Usually ':8080' is good>
$> QUEUE_NAME=<some arbitrary name. If you reuse the same name the queue state will persist as long as the redis instance does.>
$> go build src/github.com/brnsampson/redis_examples/queuer
$> ./queuer.exe (or whatever extension it has)

Windows is similar, but in powershell environment variables are defined with slightly different syntax:
$env:REDIS_ADDR = "<address of redis instance>"

To query:
Pushing a value: `curl -XPUT 127.0.0.1:8080/push -d '["dog", "cat", "frog"]'`
Popping a value: `curl 127.0.0.1:8080/pop`

MESSENGER:
For this one you want to run it in multiple terminals. When you type something into stdin on one terminal and hit enter it will appear on all terminals connected.

Config (Linux):
ON EACH TERMINAL do the following
$> REDIS_ADDR=<address of your redis instance>
$> USER_NAME=<name for this particular terminal. Make it unique for each one.>
$> CHANNEL_NAME=<common channel name. It doesn't matter what it is as long as it is the same for every terminal.>
$> go build src/github.com/brnsampson/redis_examples/messenger
$> ./messenger.exe (or whatever extension it has)

Windows is similar, but in powershell environment variables are defined with slightly different syntax:
$env:REDIS_ADDR = "<address of redis instance>"
