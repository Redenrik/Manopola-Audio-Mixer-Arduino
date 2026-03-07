//go:build darwin && cgo

package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa
#include <Cocoa/Cocoa.h>
#include <signal.h>
#include <stdlib.h>
#include <unistd.h>

@interface MAMAWindowDelegate : NSObject<NSWindowDelegate>
@property(nonatomic, assign) id<NSWindowDelegate> forwardDelegate;
@end

@interface MAMATrayDelegate : NSObject
@end

static NSStatusItem *mamaStatusItem = nil;
static NSMenu *mamaStatusMenu = nil;
static MAMATrayDelegate *mamaTrayDelegate = nil;
static MAMAWindowDelegate *mamaWindowDelegate = nil;
static NSWindow *mamaManagedWindow = nil;

@implementation MAMAWindowDelegate
- (BOOL)windowShouldClose:(id)sender {
	if (mamaManagedWindow != nil) {
		[mamaManagedWindow orderOut:nil];
	}
	return NO;
}

- (id)forwardingTargetForSelector:(SEL)aSelector {
	if ([self.forwardDelegate respondsToSelector:aSelector]) {
		return self.forwardDelegate;
	}
	return [super forwardingTargetForSelector:aSelector];
}

- (BOOL)respondsToSelector:(SEL)aSelector {
	if (aSelector == @selector(windowShouldClose:)) {
		return YES;
	}
	return [super respondsToSelector:aSelector] || [self.forwardDelegate respondsToSelector:aSelector];
}
@end

@implementation MAMATrayDelegate
- (void)onShow:(id)sender {
	if (mamaManagedWindow != nil) {
		[mamaManagedWindow makeKeyAndOrderFront:nil];
		[NSApp activateIgnoringOtherApps:YES];
	}
}

- (void)onStatusItemClick:(id)sender {
	NSEvent *event = [NSApp currentEvent];
	if (event != nil && ([event type] == NSEventTypeRightMouseDown || [event type] == NSEventTypeRightMouseUp)) {
		NSStatusBarButton *button = [mamaStatusItem button];
		if (button != nil && mamaStatusMenu != nil) {
			NSPoint menuPoint = NSMakePoint(0, NSHeight([button bounds]));
			[mamaStatusMenu popUpMenuPositioningItem:nil atLocation:menuPoint inView:button];
		}
		return;
	}
	[self onShow:sender];
}

- (void)onQuit:(id)sender {
	kill(getpid(), SIGTERM);
}
@end

static void mamaInstallWindowDelegate(void *window) {
	if (window == NULL) {
		return;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	mamaManagedWindow = w;
	if (mamaWindowDelegate == nil) {
		mamaWindowDelegate = [[MAMAWindowDelegate alloc] init];
	}
	if (mamaWindowDelegate == nil) {
		return;
	}
	if ([w delegate] != mamaWindowDelegate) {
		mamaWindowDelegate.forwardDelegate = [w delegate];
		[w setDelegate:mamaWindowDelegate];
	}
	[w setReleasedWhenClosed:NO];
}

static void mamaTrayCreate(const char *title, void *window) {
	if (window != NULL) {
		mamaInstallWindowDelegate(window);
	}
	if (mamaStatusItem != nil) {
		return;
	}

	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

	NSString *itemTitle = @"MAMA";
	if (title != NULL) {
		itemTitle = [NSString stringWithUTF8String:title];
		if (itemTitle == nil || [itemTitle length] == 0) {
			itemTitle = @"MAMA";
		}
	}

	mamaStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
	if (mamaStatusItem == nil) {
		return;
	}

	mamaTrayDelegate = [[MAMATrayDelegate alloc] init];
	if (mamaTrayDelegate == nil) {
		return;
	}

	NSMenu *menu = [[NSMenu alloc] initWithTitle:itemTitle];
	if (menu == nil) {
		return;
	}

	NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"Show MAMA" action:@selector(onShow:) keyEquivalent:@""];
	[showItem setTarget:mamaTrayDelegate];
	[menu addItem:showItem];

	[menu addItem:[NSMenuItem separatorItem]];

	NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit MAMA" action:@selector(onQuit:) keyEquivalent:@""];
	[quitItem setTarget:mamaTrayDelegate];
	[menu addItem:quitItem];

	mamaStatusMenu = menu;

	NSStatusBarButton *button = [mamaStatusItem button];
	if (button != nil) {
		[button setTitle:itemTitle];
		[button setTarget:mamaTrayDelegate];
		[button setAction:@selector(onStatusItemClick:)];
		[button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
	}
}

static void mamaTraySetIcon(const void *bytes, int length, int isTemplate) {
	if (mamaStatusItem == nil || bytes == NULL || length <= 0) {
		return;
	}
	NSData *data = [NSData dataWithBytes:bytes length:(NSUInteger)length];
	if (data == nil) {
		return;
	}
	NSImage *image = [[NSImage alloc] initWithData:data];
	if (image == nil) {
		return;
	}
	[image setSize:NSMakeSize(18, 18)];
	[image setTemplate:(isTemplate ? YES : NO)];
	NSStatusBarButton *button = [mamaStatusItem button];
	if (button != nil) {
		[button setImageScaling:NSImageScaleProportionallyDown];
		[button setImagePosition:NSImageOnly];
		[button setImage:image];
		[button setTitle:@""];
	}
}

static void mamaTrayDestroy(void) {
	if (mamaManagedWindow != nil && mamaWindowDelegate != nil && [mamaManagedWindow delegate] == mamaWindowDelegate) {
		[mamaManagedWindow setDelegate:mamaWindowDelegate.forwardDelegate];
	}
	if (mamaStatusItem != nil) {
		[[NSStatusBar systemStatusBar] removeStatusItem:mamaStatusItem];
		mamaStatusItem = nil;
	}
	mamaStatusMenu = nil;
	mamaTrayDelegate = nil;
	mamaWindowDelegate = nil;
	mamaManagedWindow = nil;
}

static void mamaWindowMiniaturize(void *window) {
	if (window == NULL) {
		return;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	[w miniaturize:nil];
}

static void mamaWindowToggleZoom(void *window) {
	if (window == NULL) {
		return;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	[w zoom:nil];
}

static int mamaWindowIsZoomed(void *window) {
	if (window == NULL) {
		return 0;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	return [w isZoomed] ? 1 : 0;
}

static void mamaWindowDrag(void *window) {
	if (window == NULL) {
		return;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	NSEvent *event = [NSApp currentEvent];
	if (event != nil) {
		[w performWindowDragWithEvent:event];
	}
}

static void mamaWindowHide(void *window) {
	if (window == NULL) {
		return;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	[w orderOut:nil];
}

static void mamaWindowShow(void *window) {
	if (window == NULL) {
		return;
	}
	NSWindow *w = (__bridge NSWindow *)window;
	[w makeKeyAndOrderFront:nil];
	[NSApp activateIgnoringOtherApps:YES];
}
*/
import "C"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"unsafe"

	webview "github.com/webview/webview_go"
)

type desktopWindowState struct {
	Desktop   bool `json:"desktop"`
	Maximized bool `json:"maximized"`
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
	window := w.Window()

	trayTitle := C.CString("MAMA")
	defer C.free(unsafe.Pointer(trayTitle))
	C.mamaTrayCreate(trayTitle, window)
	defer C.mamaTrayDestroy()
	setDarwinTrayIconIfAvailable()

	var terminated atomic.Bool
	terminate := func() {
		if terminated.CompareAndSwap(false, true) {
			w.Terminate()
		}
	}
	// macOS keeps native window chrome; do not enable custom in-page titlebar controls.
	w.Init(`window.__MAMA_DESKTOP_EMBEDDED__ = false;`)
	w.Navigate(url)

	if startHidden {
		if window != nil {
			C.mamaWindowHide(window)
		}
		fmt.Println("Desktop shell started hidden in tray on macOS.")
	} else if window != nil {
		C.mamaWindowShow(window)
	}

	go func() {
		<-ctx.Done()
		terminate()
	}()

	w.Run()
	cancel()
	return nil
}

func setDarwinTrayIconIfAvailable() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	baseDir := filepath.Dir(exePath)
	resourceDir := filepath.Clean(filepath.Join(baseDir, "..", "Resources"))
	searchDirs := []string{resourceDir, baseDir}
	candidates := []string{"mama-tray.png", "mama-app.png"}
	for _, dir := range searchDirs {
		for _, name := range candidates {
			iconPath := filepath.Join(dir, name)
			icon, err := os.ReadFile(iconPath)
			if err != nil || len(icon) == 0 {
				continue
			}
			C.mamaTraySetIcon(unsafe.Pointer(&icon[0]), C.int(len(icon)), 0)
			return
		}
	}
}
