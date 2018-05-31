.PHONY: all clean pack-darwin pack-linux oselm

TARGET := olived
OLIVE_ADDRESS ?= localhost
CLIENT_DIR_OPTS ?=

GOPATH := $(abspath $(shell pwd)/../..)
GO := GOPATH=$(GOPATH) go

all: $(TARGET).linux oselm

clean:
	-rm $(TARGET).linux
	-rm $(TARGET).darwin

.PHONY: $(TARGET).linux $(TARGET).darwin

$(TARGET).linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $@

$(TARGET).darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $@

oselm:
	cd ../oselm/quantizer; $(MAKE)

install-dep:
	$(GO) get github.com/mjibson/go-dsp/fft
	$(GO) get github.com/mjibson/go-dsp/window
	$(GO) get github.com/labstack/echo
	$(GO) get github.com/labstack/echo/middleware
	$(GO) get golang.org/x/net/websocket
	$(GO) get github.com/google/uuid
	
pack-darwin: $(TARGET).darwin
	mkdir -p $@/$(TARGET)
	mkdir -p $@/$(TARGET)/ml
	cp $(TARGET).darwin $@/$(TARGET)/$(TARGET)
	cp *.json $@/$(TARGET)
	cp ml/*.json $@/$(TARGET)/ml
	cp api.* $@/$(TARGET)
	cd $@; tar -cf ../$(TARGET).darwin.tar.gz $(TARGET)
	rm -rf $@

pack-linux: $(TARGET).linux
	mkdir -p $@/$(TARGET)
	mkdir -p $@/$(TARGET)/ml
	cp $(TARGET).linux $@/$(TARGET)/$(TARGET)
	cp *.json $@/$(TARGET)
	cp ml/*.json $@/$(TARGET)/ml
	cp api.* $@/$(TARGET)
	cd $@; tar -cf ../$(TARGET).linux.tar.gz $(TARGET)
	rm -rf $@

start-ml: test_ml_start.json
	curl -H "Content-Type:application/json" -d @$< http://$(OLIVE_ADDRESS):2223/ml/settings/set
	curl http://$(OLIVE_ADDRESS):2223/ml/start
	curl http://$(OLIVE_ADDRESS):2223/ml/settings/get

run: $(TARGET).linux
	./$(TARGET).linux

run-adcdb: $(TARGET).linux ../adc-db/adc-db
	./$(TARGET).linux -acq_config_file ./acquisition.config.adc-db.json $(CLIENT_DIR_OPTS)

run-adcdb-oselm: $(TARGET).linux ../adc-db/adc-db
	./$(TARGET).linux -acq_config_file ./acquisition.config.adc-db.json -ml_config Local $(CLIENT_DIR_OPTS)

run-adcdb-mock: $(TARGET).linux ../adc-db/adc-db
	./$(TARGET).linux -acq_config_file ./acquisition.config.adc-db-mock.json -ml_config Local -frame_record -data_dir /tmp/acqdata $(CLIENT_DIR_OPTS)

run-adcdb-oselm-debug: $(TARGET).linux ../adc-db/adc-db
	./$(TARGET).linux -acq_config_file ./acquisition.config.adc-db.json -ml_config Local -record $(CLIENT_DIR_OPTS)

run-discovery: $(TARGET).linux ../discovery-mock/discovery-mock
	./$(TARGET).linux -acq_config_file ./acquisition.config.discovery.json -ml_config Local -record $(CLIENT_DIR_OPTS)

run-adcdb-oselm-frameRec: $(TARGET).linux ../adc-db/adc-db
	./$(TARGET).linux -acq_config_file ./acquisition.config.adc-db.json -ml_config Local -frame_record -data_dir /tmp/acqdata $(CLIENT_DIR_OPTS)

run-adcdb-oselm-datafile: $(TARGET).linux ../adc-db/adc-db
	./$(TARGET).linux -acq_config_file ./acquisition.config.adc-db.json -ml_config Local -data_dir /tmp/acqdata $(CLIENT_DIR_OPTS)

../adc-db/adc-db: ../adc-db/main.go
	cd ../adc-db; go build

../discovery-mock/discovery-mock: ../discovery-mock/main.go
	cd ../discovery-mock; go build
