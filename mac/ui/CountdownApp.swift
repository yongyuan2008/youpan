import SwiftUI
import AppKit

@main
struct CountdownApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var delegate
    @StateObject(wrappedValue: CountdownStore()) private var store: CountdownStore
    @StateObject(wrappedValue: PlannerStore.shared) private var planner: PlannerStore
    @StateObject(wrappedValue: AppearanceStore()) private var appearance: AppearanceStore

    var body: some Scene {
        Window("", id: "main") {
            FittedMainViewport {
                CountdownView(store: store, planner: planner, appearance: appearance)
            }
                .background(WindowSettings(pinned: store.pinned, store: store, appearance: appearance))
        }
        .windowStyle(.hiddenTitleBar)
        .defaultSize(width: appearance.width, height: appearance.height)
    }
}

private struct FittedMainViewport<Content: View>: View {
    @ViewBuilder let content: () -> Content

    var body: some View {
        GeometryReader { geometry in
            let viewport = CGSize(width: max(1, geometry.size.width), height: max(1, geometry.size.height))
            let scale = WindowGeometry.fitScale(viewport: viewport)
            let layout = WindowGeometry.fittedLayoutSize(viewport: viewport)
            content()
                .frame(width: layout.width, height: layout.height)
                .scaleEffect(scale, anchor: .topLeading)
                .frame(width: viewport.width, height: viewport.height, alignment: .topLeading)
                .clipped()
        }
    }
}

final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    static weak var current: AppDelegate?
    weak var mainWindow: NSWindow?
    weak var appearanceStore: AppearanceStore?
    private var statusItem: NSStatusItem?
    private let desktopPanelController = DesktopCountdownPanelController()
    private var resizeSaveWorkItem: DispatchWorkItem?
    private var suppressResizePersistence = false
    private var showedResizeSaveError = false

    override init() { super.init(); Self.current = self }
    func applicationDidFinishLaunching(_ notification: Notification) { installStatusItem(); ReminderCoordinator.shared.start() }
    func applicationWillTerminate(_ notification: Notification) {
        ReminderCoordinator.shared.stop()
        desktopPanelController.destroy()
    }
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { false }
    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        let guardState = UnsavedNoteGuard.shared
        guard guardState.isDirty else { return .terminateNow }
        let alert = NSAlert(); alert.messageText = "\(guardState.editorTitle)尚未保存"; alert.informativeText = "要先保存\(guardState.editorObject)再退出有盼吗？"
        alert.addButton(withTitle: "保存并退出"); alert.addButton(withTitle: "取消"); alert.addButton(withTitle: "不保存并退出")
        switch alert.runModal() {
        case .alertFirstButtonReturn:
            if guardState.attemptSave() { return .terminateNow }
            let failure = NSAlert(); failure.messageText = "\(guardState.editorTitle)保存失败"; failure.informativeText = "已取消退出，请返回编辑器检查内容或存储文件。"; failure.runModal()
            return .terminateCancel
        case .alertThirdButtonReturn: guardState.discard(); return .terminateNow
        default: return .terminateCancel
        }
    }
    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool { openMainWindow(); return true }

    private func installStatusItem() {
        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        item.button?.image = NSImage(systemSymbolName: "calendar.badge.clock", accessibilityDescription: "有盼")
        let menu = NSMenu()
        let open = NSMenuItem(title: "打开有盼", action: #selector(openFromMenu), keyEquivalent: "o"); open.target = self; menu.addItem(open); menu.addItem(.separator())
        let quit = NSMenuItem(title: "退出有盼", action: #selector(quitFromMenu), keyEquivalent: "q"); quit.target = self; menu.addItem(quit); item.menu = menu; statusItem = item
    }
    @objc private func openFromMenu() { openMainWindow() }
    @objc private func quitFromMenu() { NSApp.terminate(nil) }
    func windowShouldClose(_ sender: NSWindow) -> Bool { sender.orderOut(nil); return false }
    func windowDidEndLiveResize(_ notification: Notification) {
        guard !suppressResizePersistence, let window = notification.object as? NSWindow,
              window === mainWindow else { return }
        scheduleWindowSizeSave(window, delay: 0.05)
    }

    func performProgrammaticResize(_ action: () -> Void) {
        resizeSaveWorkItem?.cancel()
        suppressResizePersistence = true
        action()
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) { [weak self] in self?.suppressResizePersistence = false }
    }

    func syncDesktopPanel(store: CountdownStore, appearance: AppearanceStore) {
        desktopPanelController.sync(store: store, appearance: appearance)
    }

    private func scheduleWindowSizeSave(_ window: NSWindow, delay: TimeInterval = 0.3) {
        resizeSaveWorkItem?.cancel()
        let item = DispatchWorkItem { [weak self, weak window] in
            guard let self, let window, let store = self.appearanceStore else { return }
            let actual = window.contentView?.frame.size ?? window.contentLayoutRect.size
            let size = WindowGeometry.persistedSize(actual: actual)
            do {
                try store.updateSize(width: size.width, height: size.height)
                self.showedResizeSaveError = false
            } catch {
                guard !self.showedResizeSaveError else { return }
                self.showedResizeSaveError = true
                let alert = NSAlert()
                alert.messageText = "窗口大小未能保存"
                alert.informativeText = error.localizedDescription
                alert.runModal()
            }
        }
        resizeSaveWorkItem = item
        DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: item)
    }
    private func openMainWindow() {
        NSApp.activate(ignoringOtherApps: true)
        let candidate = mainWindow ?? NSApp.windows.first(where: { $0.identifier?.rawValue == "YouPanMainWindow" })
        if let window = candidate { mainWindow = window; window.makeKeyAndOrderFront(nil) }
    }
}

final class ReminderCoordinator: NSObject, NSWindowDelegate {
    static let shared = ReminderCoordinator()
    private var timer: Timer?
    private var observers: [NSObjectProtocol] = []
    private var realPanel: NSPanel?
    private var testPanel: NSPanel?
    private var unreadEvents: [ReminderEvent] = []

    func start() {
        guard timer == nil else { return }
        timer = Timer.scheduledTimer(withTimeInterval: 20, repeats: true) { [weak self] _ in self?.check() }
        if let timer { RunLoop.main.add(timer, forMode: .common) }
        let center = NotificationCenter.default
        observers.append(center.addObserver(forName: NSApplication.didBecomeActiveNotification, object: nil, queue: .main) { [weak self] _ in self?.check() })
        observers.append(NSWorkspace.shared.notificationCenter.addObserver(forName: NSWorkspace.didWakeNotification, object: nil, queue: .main) { [weak self] _ in self?.check() })
        for name in [Notification.Name.NSSystemTimeZoneDidChange, Notification.Name.NSCalendarDayChanged] {
            observers.append(center.addObserver(forName: name, object: nil, queue: .main) { [weak self] _ in self?.check() })
        }
        check()
    }

    func stop() {
        timer?.invalidate(); timer = nil
        observers.forEach {
            NotificationCenter.default.removeObserver($0)
            NSWorkspace.shared.notificationCenter.removeObserver($0)
        }
        observers.removeAll()
        realPanel?.close(); realPanel = nil
        testPanel?.close(); testPanel = nil
        unreadEvents.removeAll()
    }

    private func check(now: Date = Date()) {
        let store = PlannerStore.shared
        let due = ReminderLogic.due(tasks: store.tasks, delivered: store.deliveredReminderKeys, now: now)
        guard !due.isEmpty else { return }
        do { try store.markRemindersDelivered(due, now: now); showReal(events: due) }
        catch { NSLog("Reminder persistence failed: %@", error.localizedDescription) }
    }

    func showTestReminder() {
        let event = ReminderEvent(taskID: "test", occurrence: "test", reminderTime: "", title: "这是一条测试提醒，不会修改任何待办。")
        let panel = testPanel ?? makePanel(title: "测试提醒")
        testPanel = panel
        render(panel: panel, events: [event], heading: "测试提醒", closeAction: #selector(dismissTest))
        positionTestPanel(panel)
        NSApp.activate(ignoringOtherApps: true); panel.makeKeyAndOrderFront(nil)
    }

    private func showReal(events: [ReminderEvent]) {
        unreadEvents = ReminderLogic.mergingUnread(unreadEvents, with: events)
        let heading = unreadEvents.count == 1 ? "待办提醒" : "\(unreadEvents.count) 项待办提醒"
        let panel = realPanel ?? makePanel(title: heading)
        realPanel = panel
        render(panel: panel, events: unreadEvents, heading: heading, closeAction: #selector(acknowledgeReal))
        panel.orderFrontRegardless()
    }

    private func makePanel(title: String) -> NSPanel {
        let panel = NSPanel(contentRect: NSRect(x: 0, y: 0, width: 410, height: 315), styleMask: [.titled, .closable, .utilityWindow], backing: .buffered, defer: false)
        panel.title = title; panel.level = .floating; panel.isReleasedWhenClosed = false
        panel.delegate = self
        panel.center()
        return panel
    }

    private func render(panel: NSPanel, events: [ReminderEvent], heading: String, closeAction: Selector) {
        panel.title = heading
        let content = NSView(frame: NSRect(x: 0, y: 0, width: 410, height: 315))
        let title = NSTextField(labelWithString: heading); title.font = .systemFont(ofSize: 18, weight: .semibold); title.frame = NSRect(x: 20, y: 270, width: 370, height: 24); content.addSubview(title)
        let text = events.map { event in event.reminderTime.isEmpty ? "• \(event.title)" : "• \(event.reminderTime)  \(event.title)" }.joined(separator: "\n\n")
        let textView = NSTextView(frame: NSRect(x: 0, y: 0, width: 350, height: 185)); textView.string = text; textView.font = .systemFont(ofSize: 13); textView.isEditable = false; textView.isSelectable = true; textView.drawsBackground = false; textView.textContainerInset = NSSize(width: 5, height: 6)
        let scroll = NSScrollView(frame: NSRect(x: 20, y: 65, width: 370, height: 190)); scroll.hasVerticalScroller = true; scroll.borderType = .bezelBorder; scroll.documentView = textView; content.addSubview(scroll)
        let close = NSButton(title: "知道了", target: self, action: closeAction); close.bezelStyle = .rounded; close.frame = NSRect(x: 310, y: 18, width: 80, height: 32); content.addSubview(close)
        panel.contentView = content
    }

    private func positionTestPanel(_ panel: NSPanel) {
        guard let realPanel, realPanel.isVisible else { return }
        var frame = panel.frame
        frame.origin = CGPoint(x: realPanel.frame.minX + 28, y: realPanel.frame.minY - 28)
        let fallback = realPanel.screen?.visibleFrame ?? NSScreen.main?.visibleFrame ?? frame
        frame = DesktopPanelGeometry.cappedFrame(frame, visibleFrames: NSScreen.screens.map(\.visibleFrame), fallback: fallback)
        panel.setFrame(frame, display: false)
    }

    func windowWillClose(_ notification: Notification) {
        guard let closingWindow = notification.object as? NSWindow else { return }
        if closingWindow === realPanel {
            unreadEvents.removeAll()
            realPanel = nil
        } else if closingWindow === testPanel {
            testPanel = nil
        }
    }

    @objc private func acknowledgeReal() { realPanel?.close() }
    @objc private func dismissTest() { testPanel?.close() }
}

private struct WindowSettings: NSViewRepresentable {
    var pinned: Bool
    @ObservedObject var store: CountdownStore
    @ObservedObject var appearance: AppearanceStore
    func makeNSView(context: Context) -> WindowObserverView { WindowObserverView() }
    func updateNSView(_ view: WindowObserverView, context: Context) {
        view.pinned = pinned
        view.store = store
        view.appearanceStore = appearance
        view.apply()
    }
    final class WindowObserverView: NSView {
        var pinned = false
        weak var store: CountdownStore?
        weak var appearanceStore: AppearanceStore?
        private var configured = false
        private var lastAppliedRequestedSize: NSSize?
        override func viewDidMoveToWindow() { super.viewDidMoveToWindow(); apply() }
        func apply() {
            guard let window, let store, let appearance = appearanceStore else { return }
            window.level = pinned ? .floating : .normal; window.title = ""; window.titleVisibility = .hidden; window.isReleasedWhenClosed = false
            window.styleMask.insert(.resizable)
            let available = availableContentSize(for: window)
            window.contentMinSize = WindowGeometry.contentMinimum(available: available)
            window.contentMaxSize = NSSize(width: AppearanceConfiguration.maximumDimension, height: AppearanceConfiguration.maximumDimension)
            window.identifier = NSUserInterfaceItemIdentifier("YouPanMainWindow"); window.delegate = AppDelegate.current
            let delegate = AppDelegate.current
            delegate?.mainWindow = window
            delegate?.appearanceStore = appearance
            delegate?.syncDesktopPanel(store: store, appearance: appearance)
            window.alphaValue = appearance.opacity
            let requested = NSSize(width: appearance.width, height: appearance.height)
            if !configured {
                configured = true
                lastAppliedRequestedSize = requested
                let target = WindowGeometry.viewportSize(requested: requested, available: available)
                performResize(using: delegate) {
                    window.setContentSize(target)
                    center(window)
                }
            } else {
                guard !window.inLiveResize, requestedSizeChanged(requested) else { return }
                lastAppliedRequestedSize = requested
                let target = WindowGeometry.viewportSize(requested: requested, available: available)
                let current = window.contentView?.frame.size ?? window.contentLayoutRect.size
                if abs(current.width - target.width) > 0.5 || abs(current.height - target.height) > 0.5 {
                    performResize(using: delegate) { window.setContentSize(target) }
                }
            }
        }

        private func requestedSizeChanged(_ requested: NSSize) -> Bool {
            guard let last = lastAppliedRequestedSize else { return true }
            return abs(last.width - requested.width) > 0.5 || abs(last.height - requested.height) > 0.5
        }

        private func availableContentSize(for window: NSWindow) -> NSSize {
            guard let screen = window.screen ?? NSScreen.main else {
                return NSSize(width: AppearanceConfiguration.minimumWidth, height: AppearanceConfiguration.minimumHeight)
            }
            return window.contentRect(forFrameRect: NSRect(origin: .zero, size: screen.visibleFrame.size)).size
        }

        private func performResize(using delegate: AppDelegate?, _ action: () -> Void) {
            if let delegate { delegate.performProgrammaticResize(action) }
            else { action() }
        }

        private func center(_ window: NSWindow) {
            guard let screen = window.screen ?? NSScreen.main else { window.center(); return }
            let visible = screen.visibleFrame
            var frame = window.frame
            frame.origin.x = visible.midX - frame.width / 2
            frame.origin.y = visible.midY - frame.height / 2
            window.setFrame(frame, display: false)
        }
    }
}
