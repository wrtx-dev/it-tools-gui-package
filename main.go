package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	r "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:it-tools-src/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

type FileLoader struct {
	http.Handler
}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (h *FileLoader) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	requestedFilename := strings.TrimPrefix(req.URL.Path, "/")
	println("Requesting file:", requestedFilename)
	if strings.HasPrefix(req.RequestURI, "/unpkg.com") {
		rres, err := http.Get("https:/" + req.RequestURI)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(fmt.Sprintf("Could not load file %s", requestedFilename)))
		}
		status := rres.StatusCode
		if status != http.StatusOK {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(fmt.Sprintf("read failed, file %s", requestedFilename)))
		}
		body, err := io.ReadAll(rres.Body)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(fmt.Sprintf("read failed, file %s", requestedFilename)))
		}
		res.Write(body)
	}
	res.WriteHeader(http.StatusBadRequest)
	res.Write([]byte(fmt.Sprintf("Could not load file %s", requestedFilename)))

}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	isMacOS := runtime.GOOS == "darwin"
	appMenu := menu.NewMenu()

	if isMacOS {
		appMenu.Append(menu.AppMenu())
		// appMenu.Append(menu.EditMenu())
		// appMenu.Append(menu.WindowMenu())
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "it-tools",
		Width:  1280,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: NewFileLoader(),
		},
		Menu:                     appMenu,
		EnableDefaultContextMenu: true,
		BackgroundColour:         &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:                app.startup,
		OnDomReady: func(ctx context.Context) {
			r.WindowExecJS(ctx, `
				document.addEventListener("click", function(ev){
				    const anchor = ev.target.closest('a');
				    if (anchor && anchor.href && /^https?:\/\//i.test(anchor.href)){
				        ev.preventDefault();
				        console.log("Intercepted link:", anchor.href);
				        window.runtime.BrowserOpenURL(anchor.href);
				    }
				})
				window.origin_fetch = window.fetch;
				window.fetch= function(input, init){
    				console.log("input type:", typeof input)
    				let url = input instanceof Request ? input.url : input;
    				try {
    				    const parsedUrl = new URL(url);
    				    console.log("Request Host:", parsedUrl.hostname);
    				}catch(e){
    				    console.log("Invalid URL:", url, typeof url)
						if (typeof input === "string" && input.startsWith("//unpkg.com")) {
							input = input.slice(1);
						}
    				}
    				return window.origin_fetch(input, init);
				}
				`)
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title: "it-tools gui",
				Message: `
				it-tools: https://github.com/CorentinTh/it-tools`,
				Icon: icon,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Appearance:           mac.DefaultAppearance,
		},
		HideWindowOnClose: isMacOS,

		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
