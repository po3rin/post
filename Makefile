
contents:
	postctl contents -p "https://github.com/po3rin/post/tree/master"

lint:
	postctl contents -p "https://github.com/po3rin/post/tree/master" -l

sync:
	postctl sync -u http://localhost:8081/post

sync-agent:
	postctl sync -u http://localhost:8081/post --agent-mode
