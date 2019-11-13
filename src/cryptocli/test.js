function start() {
	pipe("tcp-server --listen :8080", undefined, {"max-concurrent-streams": 10, "multi-streams": true})
}

function callback() {
	for (;;) {
		var message = readline()
		if (message === undefined) {
			return
		}
		log(message)

		write(message + "\n")
	}
}
