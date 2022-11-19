cslauncher:
	CGO_ENABLED=1 GOARCH=amd64  go build -ldflags "-s -w" -trimpath -o cslauncher_amd64 main.go
	CGO_ENABLED=1 GOARCH=arm64  go build -ldflags "-s -w" -trimpath -o cslauncher_arm64 main.go
	upx cslauncher_amd64
	lipo -create cslauncher_amd64 cslauncher_arm64  -output cslauncher
	rm cslauncher_amd64 cslauncher_arm64
	
	mkdir build/cslauncher.app
	mkdir build/cslauncher.app/Contents
	mkdir build/cslauncher.app/Contents/MacOS
	mkdir build/cslauncher.app/Contents/Resources
	
	cp build/Info.plist.src build/cslauncher.app/Contents/Info.plist
	mv cslauncher build/cslauncher.app/Contents/MacOS/cslauncher
	cp build/Appicon.icns build/cslauncher.app/Contents/Resources/Appicon.icns
clean:
	rm cslauncher_amd64 cslauncher_arm64 cslauncher
	rm -rf build/cslauncher.app