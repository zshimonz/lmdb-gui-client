.PHONY: build_macos build_windows build_linux

build_macos:
	go mod tidy
	go install fyne.io/fyne/v2/cmd/fyne@latest
	cp FyneApp.toml fyne-app.toml
	fyne package -os darwin; mv -f fyne-app.toml FyneApp.toml

build_windows:
	go mod tidy
	go install fyne.io/fyne/v2/cmd/fyne@latest
	cp FyneApp.toml fyne-app.toml
	fyne package -os windows; mv -f fyne-app.toml FyneApp.toml

build_linux:
	go mod tidy
	go install fyne.io/fyne/v2/cmd/fyne@latest
	cp FyneApp.toml fyne-app.toml
	fyne package -os linux; mv -f fyne-app.toml FyneApp.toml