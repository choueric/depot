PREFIX := depot
LOCAL := $(GOPATH)/bin/$(PREFIX)-local
SERVER := $(GOPATH)/bin/$(PREFIX)-server

all: $(LOCAL) $(SERVER)

.PHONY: clean

clean:
	rm -f $(LOCAL) $(SERVER) $(TEST)

# -a option is needed to ensure we disabled CGO
$(LOCAL): *.go $(PREFIX)-local/*.go
	cd $(PREFIX)-local; go install

$(SERVER): *.go $(PREFIX)-server/*.go
	cd $(PREFIX)-server; go install

local: $(LOCAL)

server: $(SERVER)

test:
	go test

curl:
	curl --socks5 127.0.0.1:8864 --proxy-user user:password www.baidu.com
