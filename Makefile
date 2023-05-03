all:
	wails build -clean -platform darwin/arm64
	wails build -platform darwin/amd64
	wails build -platform windows/amd64
	wails build -platform linux/amd64
	wails build -platform linux/arm64
