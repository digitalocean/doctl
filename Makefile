build: .deps test bin

bin: doctl.go droplet.go sshkey.go
	gox -arch="386 amd64" -os="darwin linux windows" -output="./bin/{{.OS}}_{{.Arch}}/doctl" .

.deps:
	touch .deps
	godep restore

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin
	rm .deps