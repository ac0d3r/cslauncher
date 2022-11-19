package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

const (
	csjar           = "cobaltstrike.jar"
	csexe           = "cobaltstrike"
	defualtStartCmd = "java -XX:ParallelGCThreads=4 -XX:+AggressiveHeap -XX:+UseParallelGC -jar cobaltstrike.jar $*"

	cnfName = "cslauncher"
)

var (
	ErrMissingJar     = errors.New("missing cobaltstrike.jar file")
	ErrAlreadyRunning = errors.New("already running")
)

//go:embed Appicon.png
var icon []byte

var Config = func() *config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir, _ = os.Getwd()
	}

	homeDir = path.Join(homeDir, ".config", cnfName)
	return &config{
		homeDir:    homeDir,
		configFile: ".cslauncher",
		config:     make(map[string]string)}
}()

type config struct {
	homeDir    string
	configFile string

	config map[string]string
}

func (c *config) Init() error {
	var data []byte
	exist, err := pathIsExist(c.homeDir)
	if err != nil {
		return err
	}
	if !exist {
		if err := os.Mkdir(c.homeDir, os.ModePerm); err != nil {
			return err
		}
	}

	cnf := path.Join(c.homeDir, c.configFile)
	exist, err = pathIsExist(cnf)
	if err != nil {
		return err
	}
	if !exist {
		f, err := os.Create(cnf)
		if err != nil {
			return err
		}
		f.Close()
	} else if data, err = os.ReadFile(cnf); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &c.config)
}

func (c *config) Save() error {
	if c.config == nil {
		return nil
	}
	data, err := json.Marshal(c.config)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(c.homeDir, c.configFile), data, os.ModePerm)
}

func (c *config) Set(k, v string) {
	c.config[strings.ToLower(k)] = v
}

func (c *config) Get(k string) string {
	return c.config[strings.ToLower(k)]
}

func main() {
	app := newApp()
	systray.Run(app.startup, app.shutdown)
}

type app struct {
	csCmdArgs string
	cspath    string
}

func newApp() *app {
	return &app{}
}

func (a *app) selectPath(paths string) error {
	if len(paths) == 0 {
		return nil
	}
	files, err := os.ReadDir(paths)
	if err != nil {
		return err
	}
	findJar := false
	for _, f := range files {
		if f.Name() == csexe {
			a.csCmdArgs = fmt.Sprintf("/bin/bash %s", path.Join(paths, f.Name()))
		}
		if f.Name() == csjar {
			findJar = true
		}
	}
	if !findJar {
		return ErrMissingJar
	}
	if len(a.csCmdArgs) == 0 {
		a.csCmdArgs = defualtStartCmd
	}
	a.cspath = paths
	Config.Set("cs", a.cspath)
	Config.Set("csCmdArgs", a.csCmdArgs)
	return nil
}

func (a *app) showinFinder() error {
	if len(a.cspath) == 0 {
		return os.ErrNotExist
	}
	return exec.Command("open", a.cspath).Run()
}

func (a *app) startCS() error {
	if len(a.csCmdArgs) == 0 {
		return errors.New("must set cobaltstrike path")
	}
	args := strings.Split(a.csCmdArgs, " ")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = a.cspath
	return cmd.Start()
}

func (a *app) startup() {
	systray.SetIcon(icon)
	systray.SetTitle("cslauncher")
	systray.SetTooltip("Cobalt Strike Launcher on macOS - zznQ")

	pathShowMenu := systray.AddMenuItem("Path: ", "")
	pathShowMenu.Disable()
	cmdArgsMenu := systray.AddMenuItem("CmdArgs: ", "")
	systray.AddSeparator()
	selectPathMenu := systray.AddMenuItem("Select Path", "")
	showinFinderMenu := systray.AddMenuItem("ShowInFinder", "")
	startCSMenu := systray.AddMenuItem("Start CS", "")
	systray.AddSeparator()
	quitMenu := systray.AddMenuItem("Quit", "")

	// config
	if err := Config.Init(); err != nil {
		zenity.Error(err.Error(),
			zenity.Title("Error"),
			zenity.ErrorIcon)
	}
	a.cspath = Config.Get("cs")
	a.csCmdArgs = Config.Get("csCmdArgs")
	if len(a.cspath) != 0 {
		pathShowMenu.SetTitle(fmt.Sprintf("Path: %s", a.cspath))
	}
	if len(a.csCmdArgs) != 0 {
		cmdArgsMenu.SetTitle(fmt.Sprintf("CmdArgs: %s", a.csCmdArgs))
	}

	go func() {
		for {
			select {
			case <-cmdArgsMenu.ClickedCh:
				cmdargs, err := zenity.Entry("set cs start command args", zenity.EntryText(a.csCmdArgs))
				if !errors.Is(err, zenity.ErrCanceled) && err != nil {
					zenity.Error(err.Error(),
						zenity.Title("Error"),
						zenity.ErrorIcon)
				}
				if len(cmdargs) != 0 {
					a.csCmdArgs = cmdargs
					cmdArgsMenu.SetTitle(fmt.Sprintf("CmdArgs: %s", a.csCmdArgs))
					Config.Set("csCmdArgs", a.csCmdArgs)
				}
			case <-selectPathMenu.ClickedCh:
				path, err := zenity.SelectFile(zenity.Directory())
				if !errors.Is(err, zenity.ErrCanceled) && err != nil {
					zenity.Error(err.Error(),
						zenity.Title("Error"),
						zenity.ErrorIcon)
				}

				if err := a.selectPath(path); err != nil {
					zenity.Error(err.Error(),
						zenity.Title("Error"),
						zenity.ErrorIcon)
				} else {
					pathShowMenu.SetTitle(fmt.Sprintf("Path: %s", a.cspath))
					cmdArgsMenu.SetTitle(fmt.Sprintf("CmdArgs: %s", a.csCmdArgs))
				}
			case <-showinFinderMenu.ClickedCh:
				if err := a.showinFinder(); err != nil {
					zenity.Error(err.Error(),
						zenity.Title("Error"),
						zenity.ErrorIcon)
				}
			case <-startCSMenu.ClickedCh:
				err := a.startCS()
				if err != nil {
					zenity.Error(err.Error(),
						zenity.Title("Error"),
						zenity.ErrorIcon)
				}
			case <-quitMenu.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (a *app) shutdown() {
	Config.Save()
}

func pathIsExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err == nil {
		return true, nil
	} else {
		return false, err
	}
}
