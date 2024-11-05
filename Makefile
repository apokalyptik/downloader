BUILD_VERSION=1
BUILD_TIME=$(shell date +%s) 

all: release

fyne:
	go install github.com/fyne-io/fyne-cross@latest
	go install fyne.io/fyne/v2/cmd/fyne@v2.5

release:
	@rm -rf fyne-cross
	fyne-cross darwin -app-id com.automattic.downloader -app-version $(BUILD_VERSION).$(BUILD_TIME) -arch=* -pull
	fyne-cross windows -app-id com.automattic.downloader -app-version $(BUILD_VERSION).$(BUILD_TIME) -arch=* -pull
	rm -rf fyne-cross/bin fyne-cross/tmp
