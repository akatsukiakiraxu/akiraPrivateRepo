ifdef PETALINUX
include apps.common.mk
endif

APP = adc-dma

CFLAGS += -std=c99 -O2 -I./include
LDFLAGS += -pthread


# Add any other object files to this list below

APP_OBJS = adc-db.o
ifdef PETALINUX
APP_OBJS += async-io.o summary.o sensor.o
LDFLAGS += -L./lib/petalinux
else
CPPFLAGS += -std=c++11 -O2 -I./include
APP_OBJS += StreamingAI.o PropertyUtility.o AsyncControlImpl.o AsyncCommand.o ChangeSettingUtility.o
LDLIBS += -lbiodaq -lstdc++ -lm -lboost_system
LDFLAGS += -L./lib/x86
endif

#APP_OBJS = adc-db.o  dummy-sensor.o

LDLIBS += -lpaho-mqtt3c

all: build

build: $(APP)

$(APP): $(APP_OBJS)
	$(CC) $(LDFLAGS) -o $@ $(APP_OBJS) $(LDLIBS)

clean:
	-rm -f $(APP) *.elf *.gdb *.o

.PHONY: install image

ifdef PETALINUX
install: $(APP)
	$(TARGETINST) -d $(APP) /bin/$(APP)
else
install:
	ln -sf $(shell pwd)/$(APP) /bin/$(APP)
endif

%.o: %.c
	$(CC) -c $(CFLAGS) -o $@ $<

%.o: %.cpp
	$(CXX) -c $(CPPFLAGS) -o $@ $<

help:
	@echo ""
	@echo "Quick reference for various supported build targets for $(INSTANCE)."
	@echo "----------------------------------------------------"
	@echo "  clean                  clean out build objects"
	@echo "  all                    build $(INSTANCE) and install to rootfs host copy"
	@echo "  build                  build subsystem"
	@echo "  install                install built objects to rootfs host copy"
