//go:build darwin && cgo

#import <Cocoa/Cocoa.h>
#include "_cgo_export.h"
#include "native_darwin.h"

static const CGFloat barWidth = 72, barHeight = 9, barRadius = 2, menuTabStop = 230;

@interface PomoDelegate : NSObject <NSApplicationDelegate>
@property (strong) NSStatusItem *statusItem;
@property (strong) NSMenu *menu;
@property (strong) NSMutableDictionary<NSNumber *, NSMenuItem *> *menuItems;
@end

static PomoDelegate *delegate;

static NSColor *hexColor(int rgb) {
	return [NSColor colorWithSRGBRed:((rgb >> 16) & 0xff) / 255.0
	                           green:((rgb >> 8) & 0xff) / 255.0
	                            blue:(rgb & 0xff) / 255.0
	                           alpha:1];
}

// a miniature of the TUI progress bar: the gradient spans the whole
// track and is clipped to the completed part. Colors resolve at draw
// time, so the track follows light and dark menu bars.
static NSImage *barImage(double progress, bool paused) {
	return [NSImage imageWithSize:NSMakeSize(barWidth, barHeight) flipped:NO drawingHandler:^BOOL(NSRect rect) {
		[[[NSColor labelColor] colorWithAlphaComponent:0.15] setFill];
		[[NSBezierPath bezierPathWithRoundedRect:rect xRadius:barRadius yRadius:barRadius] fill];

		NSRect filled = rect;
		filled.size.width = round(rect.size.width * progress);
		if (filled.size.width <= 0) {
			return YES;
		}

		[NSGraphicsContext saveGraphicsState];
		CGContextSetAlpha([NSGraphicsContext currentContext].CGContext, paused ? 0.45 : 1);
		[[NSBezierPath bezierPathWithRoundedRect:filled xRadius:barRadius yRadius:barRadius] addClip];
		NSGradient *gradient = [[NSGradient alloc] initWithStartingColor:hexColor(0x5A56E0)
		                                                     endingColor:hexColor(0xEE6FF8)];
		[gradient drawInRect:rect angle:0];
		[NSGraphicsContext restoreGraphicsState];
		return YES;
	}];
}

@implementation PomoDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
	self.statusItem.visible = YES;

	NSStatusBarButton *button = self.statusItem.button;
	button.imagePosition = NSImageRight;
	button.target = self;
	button.action = @selector(clicked:);
	[button sendActionOn:NSEventMaskLeftMouseDown | NSEventMaskRightMouseUp];

	goReady();
}

- (void)clicked:(id)sender {
	NSEvent *event = NSApp.currentEvent;
	bool secondary = event.type == NSEventTypeRightMouseUp ||
		(event.modifierFlags & NSEventModifierFlagControl);

	if (secondary) {
		goRightClick();
	} else {
		goLeftClick();
	}
}

- (void)menuAction:(NSMenuItem *)item {
	goMenuAction((int)item.tag);
}

@end

void pomoRun(void) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
		delegate = [[PomoDelegate alloc] init];
		NSApp.delegate = delegate;
		[NSApp run];
	}
}

void pomoQuit(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp terminate:nil];
	});
}

// may be called from any thread
void pomoSetStatus(const char *label, double progress, bool paused) {
	NSString *title = [NSString stringWithUTF8String:label];

	dispatch_async(dispatch_get_main_queue(), ^{
		NSStatusBarButton *button = delegate.statusItem.button;
		if (button == nil) {
			return;
		}

		NSDictionary *attributes = @{
			NSFontAttributeName: [NSFont monospacedSystemFontOfSize:12 weight:NSFontWeightRegular],
			NSForegroundColorAttributeName: paused ? [NSColor secondaryLabelColor] : [NSColor labelColor],
			NSBaselineOffsetAttributeName: @(-0.5),
		};
		button.attributedTitle = [[NSAttributedString alloc] initWithString:[title stringByAppendingString:@" "]
		                                                         attributes:attributes];
		button.image = barImage(progress, paused);
	});
}

// menu building runs on the main thread, inside goLeftClick

void pomoMenuBegin(void) {
	delegate.menu = [[NSMenu alloc] init];
	delegate.menu.autoenablesItems = NO;
	delegate.menuItems = [NSMutableDictionary dictionary];
}

void pomoMenuAdd(int parent, int itemID, const char *title, const char *detail,
                        bool enabled, bool checked, bool separator, bool hasAction) {
	NSMenu *menu = delegate.menu;
	if (parent >= 0) {
		NSMenuItem *parentItem = delegate.menuItems[@(parent)];
		if (parentItem.submenu == nil) {
			parentItem.submenu = [[NSMenu alloc] init];
			parentItem.submenu.autoenablesItems = NO;
		}
		menu = parentItem.submenu;
	}

	if (separator) {
		[menu addItem:[NSMenuItem separatorItem]];
		return;
	}

	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:[NSString stringWithUTF8String:title]
	                                              action:nil
	                                       keyEquivalent:@""];
	item.enabled = enabled;
	item.state = checked ? NSControlStateValueOn : NSControlStateValueOff;
	item.tag = itemID;

	if (hasAction) {
		item.target = delegate;
		item.action = @selector(menuAction:);
	}

	if (strlen(detail) > 0) {
		NSFont *font = [NSFont menuFontOfSize:0];
		NSMutableParagraphStyle *style = [[NSMutableParagraphStyle alloc] init];
		style.tabStops = @[[[NSTextTab alloc] initWithTextAlignment:NSTextAlignmentRight location:menuTabStop options:@{}]];

		NSMutableAttributedString *text = [[NSMutableAttributedString alloc] initWithString:item.title attributes:@{
			NSFontAttributeName: font,
			NSForegroundColorAttributeName: [NSColor labelColor],
			NSParagraphStyleAttributeName: style,
		}];
		[text appendAttributedString:[[NSAttributedString alloc]
			initWithString:[@"\t" stringByAppendingString:[NSString stringWithUTF8String:detail]]
			    attributes:@{
				NSFontAttributeName: font,
				NSForegroundColorAttributeName: [NSColor secondaryLabelColor],
				NSParagraphStyleAttributeName: style,
			}]];
		item.attributedTitle = text;
	}

	delegate.menuItems[@(itemID)] = item;
	[menu addItem:item];
}

void pomoMenuShow(void) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
	// still the simplest way to show a menu for one click type only
	[delegate.statusItem popUpStatusItemMenu:delegate.menu];
#pragma clang diagnostic pop
}
