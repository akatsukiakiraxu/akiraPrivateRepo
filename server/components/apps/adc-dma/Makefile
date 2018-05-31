ifndef PETALINUX
$(error "Error: PETALINUX environment variable not set.  Change to the root of your PetaLinux install, and source the settings.sh file")
endif

include apps.common.mk

all: build install

build: adc-dma

adc-dma: adc-dma.c adc-dma.h
	$(CC) -std=gnu99 -pthread -O2 $(CFLAGS) $(LDFLAGS) -o $@ $< $(LDLIBS)

clean:
	$(RM) adc-dma

.PHONY: install image

install:
	$(TARGETINST) -d adc-dma /bin/adc-dma

help:
	@echo ""
	@echo "Quick reference for various supported build targets for $(INSTANCE)."
	@echo "----------------------------------------------------"
	@echo "  all                    build $(INSTANCE) and install to rootfs host copy"
	@echo "  install                install built objects to rootfs host copy"
