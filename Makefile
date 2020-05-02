new:
	postctl new

contents:
	postctl contents -p "https://github.com/po3rin/post/tree/master"

lint:
	postctl contents -p "https://github.com/po3rin/post/tree/master" -l

sync:
	postctl sync -u http://localhost:8081/post -a
