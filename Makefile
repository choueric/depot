PREFIX := depot
LOCAL := $(GOPATH)/bin/$(PREFIX)-local
SERVER := $(GOPATH)/bin/$(PREFIX)-server

DIR:=$(HOME)/.depot

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

# ncat is a tool in nmap and should at least above version 7 to support socks5
ssh:
	ssh -o ProxyCommand='ncat --proxy 127.0.0.1:8864 --proxy-type socks5 \
		--proxy-auth user:password %h %p' 127.0.0.1 -p 22

install:
	install -d $(DIR)/js $(DIR)/css
	cd $(DIR); rm -rf *.html js css
	cd $(PREFIX)-server; cp -v -a *.html js css $(DIR)
