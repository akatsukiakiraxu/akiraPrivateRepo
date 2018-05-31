ifndef PETALINUX
$(error "Error: PETALINUX environment variable not set.  Change to the root of your PetaLinux install, and source the settings.sh file")
endif

include apps.common.mk

all: install

clean:
	-rm -f *.elf *.gdb *.o

.PHONY: install image

install:
	$(TARGETINST) -d frontend/ /srv/www/frontend
	$(TARGETINST) -d bower_components/ /srv/www/bower_components

help:
	@echo ""
	@echo "Quick reference for various supported build targets for $(INSTANCE)."
	@echo "----------------------------------------------------"
	@echo "  clean                  clean out build objects"
	@echo "  all                    build $(INSTANCE) and install to rootfs host copy"
	@echo "  build                  build subsystem"
	@echo "  install                install built objects to rootfs host copy"
