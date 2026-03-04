//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	webview "github.com/jchv/go-webview2"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	trayCallbackMessage = win.WM_APP + 0x52
	trayMenuShowID      = 1001
	trayMenuQuitID      = 1002
)

var (
	desktopWndProcPtr  uintptr
	desktopWndProcOnce sync.Once
	desktopShells      sync.Map // map[win.HWND]*desktopShell
)

type desktopShell struct {
	web             webview.WebView
	hwnd            win.HWND
	cancel          context.CancelFunc
	originalWndProc uintptr
	menu            win.HMENU
	trayData        win.NOTIFYICONDATA
	quitOnce        sync.Once
}

func defaultDesktopMode() bool {
	return true
}

func runDesktopShell(ctx context.Context, cancel context.CancelFunc, url string, _ bool, startHidden bool) error {
	w := webview.New(false)
	if w == nil {
		return fmt.Errorf("failed to initialize desktop webview")
	}
	defer w.Destroy()

	w.SetTitle("MAMA Audio Mixer")
	w.SetSize(1220, 840, webview.HintNone)
	w.Navigate(url)

	hwnd := win.HWND(uintptr(w.Window()))
	if hwnd == 0 {
		return fmt.Errorf("desktop window handle not available")
	}

	shell, err := newDesktopShell(hwnd, w, cancel)
	if err != nil {
		return err
	}
	defer shell.dispose()
	if startHidden {
		shell.hide()
	} else {
		shell.show()
	}

	go func() {
		<-ctx.Done()
		w.Terminate()
	}()

	w.Run()
	return nil
}

func newDesktopShell(hwnd win.HWND, w webview.WebView, cancel context.CancelFunc) (*desktopShell, error) {
	menu, err := createTrayMenu()
	if err != nil {
		return nil, err
	}

	s := &desktopShell{
		web:    w,
		hwnd:   hwnd,
		cancel: cancel,
		menu:   menu,
	}

	trayIcon := loadIcon(16, 16, []string{"mama-tray.ico", "mama-app.ico"})
	if trayIcon == 0 {
		trayIcon = win.LoadIcon(0, win.MAKEINTRESOURCE(win.IDI_APPLICATION))
	}
	if trayIcon == 0 {
		win.DestroyMenu(menu)
		return nil, fmt.Errorf("failed to load tray icon")
	}

	windowIcon := loadIcon(256, 256, []string{"mama-app.ico", "mama-tray.ico"})
	if windowIcon == 0 {
		windowIcon = trayIcon
	}
	win.SendMessage(hwnd, win.WM_SETICON, 1, uintptr(windowIcon))
	win.SendMessage(hwnd, win.WM_SETICON, 0, uintptr(windowIcon))

	nid := win.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(win.NOTIFYICONDATA{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           win.NIF_MESSAGE | win.NIF_ICON | win.NIF_TIP,
		UCallbackMessage: trayCallbackMessage,
		HIcon:            trayIcon,
	}
	copyTrayString(&nid.SzTip, "MAMA Audio Mixer")
	if !win.Shell_NotifyIcon(win.NIM_ADD, &nid) {
		win.DestroyMenu(menu)
		return nil, fmt.Errorf("failed to create tray icon")
	}
	s.trayData = nid

	desktopShells.Store(hwnd, s)
	oldProc := win.SetWindowLongPtr(hwnd, win.GWLP_WNDPROC, getDesktopWndProcPtr())
	s.originalWndProc = oldProc

	return s, nil
}

func (s *desktopShell) dispose() {
	desktopShells.Delete(s.hwnd)

	if s.originalWndProc != 0 {
		win.SetWindowLongPtr(s.hwnd, win.GWLP_WNDPROC, s.originalWndProc)
	}

	if s.trayData.HWnd != 0 {
		win.Shell_NotifyIcon(win.NIM_DELETE, &s.trayData)
	}
	if s.menu != 0 {
		win.DestroyMenu(s.menu)
	}
}

func (s *desktopShell) wndProc(msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case trayCallbackMessage:
		s.onTrayMessage(uint32(lParam))
		return 0
	case win.WM_CLOSE:
		s.hide()
		return 0
	}

	if s.originalWndProc != 0 {
		return win.CallWindowProc(s.originalWndProc, s.hwnd, msg, wParam, lParam)
	}
	return win.DefWindowProc(s.hwnd, msg, wParam, lParam)
}

func (s *desktopShell) onTrayMessage(msg uint32) {
	switch msg {
	case win.WM_LBUTTONDBLCLK:
		s.show()
	case win.WM_RBUTTONUP, win.WM_CONTEXTMENU:
		s.showTrayMenu()
	}
}

func (s *desktopShell) showTrayMenu() {
	var cursor win.POINT
	if !win.GetCursorPos(&cursor) {
		return
	}
	win.SetForegroundWindow(s.hwnd)
	cmd := win.TrackPopupMenu(s.menu, win.TPM_RETURNCMD|win.TPM_RIGHTBUTTON, cursor.X, cursor.Y, 0, s.hwnd, nil)
	switch cmd {
	case trayMenuShowID:
		s.show()
	case trayMenuQuitID:
		s.quit()
	}
	win.PostMessage(s.hwnd, win.WM_NULL, 0, 0)
}

func (s *desktopShell) show() {
	win.ShowWindow(s.hwnd, win.SW_RESTORE)
	win.ShowWindow(s.hwnd, win.SW_SHOW)
	win.SetForegroundWindow(s.hwnd)
}

func (s *desktopShell) hide() {
	win.ShowWindow(s.hwnd, win.SW_HIDE)
}

func (s *desktopShell) quit() {
	s.quitOnce.Do(func() {
		s.cancel()
		s.web.Terminate()
	})
}

func getDesktopWndProcPtr() uintptr {
	desktopWndProcOnce.Do(func() {
		desktopWndProcPtr = syscall.NewCallback(func(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
			if shell, ok := desktopShells.Load(hwnd); ok {
				return shell.(*desktopShell).wndProc(msg, wParam, lParam)
			}
			return win.DefWindowProc(hwnd, msg, wParam, lParam)
		})
	})
	return desktopWndProcPtr
}

func createTrayMenu() (win.HMENU, error) {
	menu := win.CreatePopupMenu()
	if menu == 0 {
		return 0, fmt.Errorf("CreatePopupMenu failed")
	}

	if !appendMenuString(menu, trayMenuShowID, "Show MAMA") {
		win.DestroyMenu(menu)
		return 0, fmt.Errorf("failed to add tray Show action")
	}
	if !appendMenuSeparator(menu) {
		win.DestroyMenu(menu)
		return 0, fmt.Errorf("failed to add tray separator")
	}
	if !appendMenuString(menu, trayMenuQuitID, "Quit MAMA") {
		win.DestroyMenu(menu)
		return 0, fmt.Errorf("failed to add tray Quit action")
	}

	return menu, nil
}

func appendMenuString(menu win.HMENU, itemID uint32, text string) bool {
	ptr, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return false
	}
	return win.InsertMenuItem(menu, uint32(win.GetMenuItemCount(menu)), true, &win.MENUITEMINFO{
		CbSize:     uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
		FMask:      win.MIIM_STRING | win.MIIM_ID,
		WID:        itemID,
		DwTypeData: ptr,
		Cch:        uint32(len(text)),
	})
}

func appendMenuSeparator(menu win.HMENU) bool {
	return win.InsertMenuItem(menu, uint32(win.GetMenuItemCount(menu)), true, &win.MENUITEMINFO{
		CbSize: uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
		FMask:  win.MIIM_FTYPE,
		FType:  win.MFT_SEPARATOR,
	})
}

func loadIcon(width, height int32, fileNames []string) win.HICON {
	exePath, err := os.Executable()
	if err != nil {
		return 0
	}
	baseDir := filepath.Dir(exePath)

	for _, name := range fileNames {
		path := filepath.Join(baseDir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		ptr, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		icon := win.HICON(win.LoadImage(0, ptr, win.IMAGE_ICON, width, height, win.LR_LOADFROMFILE))
		if icon != 0 {
			return icon
		}
	}
	return 0
}

func copyTrayString(dst *[128]uint16, value string) {
	utf16Value, err := windows.UTF16FromString(value)
	if err != nil {
		return
	}
	copy(dst[:], utf16Value)
}
